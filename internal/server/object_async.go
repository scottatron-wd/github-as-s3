package server

import (
	"errors"
	"fmt"
	"github-as-s3/internal/git"
	"github-as-s3/internal/s3"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

func (h *Handler) HeadObjectAsync(c echo.Context) error {
	bucketName := c.Param("bucket")
	objectKey := c.Param("*")
	versionID := c.QueryParam("versionId")

	logger := log.Ctx(c.Request().Context()).With().
		Str("bucket", bucketName).
		Str("object", objectKey).
		Str("versionId", versionID).
		Str("command", "HeadObject").
		Bool("async", true).
		Logger()

	logger.Debug().Msg("HeadObject request received")
	file, commit, err := h.gitasync.Head(c.Request().Context(), bucketName, objectKey, versionID)
	if err != nil || file == nil || commit == nil {
		return h.s3ErrorResponse(c, http.StatusNotFound, "NoSuchKey", "The specified key does not exist.", objectKey)
	}

	lastModified := commit.Committer.When

	logger.Debug().
		Str("file_sha", file.Hash.String()).
		Int64("size", file.Size).
		Str("type", file.Type().String()).
		Time("last_modified", lastModified).
		Msg("Object found, setting headers")

	c.Response().Header().Set("ETag", fmt.Sprintf("\"%s\"", file.Hash.String()))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", file.Size))

	c.Response().Header().Set("x-amz-version-id", commit.Hash.String())

	contentType := mime.TypeByExtension(path.Ext(objectKey))
	if contentType == "" {
		contentType = "application/octet-stream" // S3 default
	}
	c.Response().Header().Set("Content-Type", contentType)
	c.Response().Header().Set("Accept-Ranges", "bytes") // Common for S3 objects
	c.Response().Header().Set("Last-Modified", lastModified.Format(http.TimeFormat))

	return c.NoContent(http.StatusOK)
}

func (h *Handler) PutObjectAsync(c echo.Context) error {
	ctx := c.Request().Context()

	// Check if this is a CopyObject operation
	copySource := c.Request().Header.Get("x-amz-copy-source")
	if copySource != "" {
		return h.CopyObjectAsync(c)
	}

	logger := log.Ctx(ctx).With().Str("command", "PutObject").Bool("async", true).Logger()

	logger.Debug().Msg("PutObject.Start")

	bucketName := c.Param("bucket")
	if bucketName == "" {
		logger.Warn().Msg("Bucket name is missing")
		return c.String(http.StatusBadRequest, "Bucket name is missing")
	}

	logger = logger.With().Str("bucket", bucketName).Logger()

	objectKey := c.Param("*")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}

	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing after trimming prefix")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}

	logger.Debug().Str("bucket", bucketName).Str("key", objectKey).Msg("Parsed parameters")

	// async difference starts
	repo, err := h.gitasync.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		return c.String(http.StatusInternalServerError, "Failed to clone repository")
	}
	logger.Debug().Msg("Cloned repository")

	err = h.gitasync.PutRaw(ctx, repo, bucketName, objectKey, c.Request().Body)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to put object")
		return c.String(http.StatusInternalServerError, "Failed to put object")
	}

	var commitSHA string
	headRef, err := repo.Head()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get repository HEAD for versioning")
		// it's ok to not have a commit SHA
	}

	commitSHA = headRef.Hash().String()
	logger.Debug().Str("commitSHA", commitSHA).Msg("Got commit SHA for version ID")

	c.Response().Header().Set("ETag", fmt.Sprintf(`"%s"`, commitSHA)) // ETag should be quoted

	if commitSHA != "" {
		c.Response().Header().Set("x-amz-version-id", commitSHA)
	}

	logger.Info().Str("versionId", commitSHA).Msg("PutObject.OK")
	return c.String(http.StatusOK, "")
}

func (h *Handler) DeleteObjectAsync(c echo.Context) error {
	ctx := c.Request().Context()
	logger := log.Ctx(ctx).With().Str("command", "DeleteObject").Logger()

	logger.Debug().Msg("DeleteObject.Start")

	bucketName := c.Param("bucket")
	if bucketName == "" {
		logger.Warn().Msg("Bucket name is missing")
		return c.String(http.StatusBadRequest, "Bucket name is missing")
	}
	logger = logger.With().Str("bucket", bucketName).Logger()

	objectKey := c.Param("*")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}
	objectKey = strings.TrimPrefix(objectKey, "/")
	if objectKey == "" {
		logger.Warn().Msg("Object key is missing after trimming prefix")
		return c.String(http.StatusBadRequest, "Object key is missing")
	}
	logger = logger.With().Str("key", objectKey).Logger()

	logger.Debug().Msg("Parsed parameters")

	repo, err := h.gitasync.Clone(ctx, bucketName)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to clone repository")
		// could make this stricter if we wanted: return c.String(http.StatusNotFound, fmt.Sprintf("Bucket (repository) '%s' not found: %v", bucketName, err))
		c.Response().Header().Set("x-amz-delete-marker", "true")
		logger.Info().Msg("Bucket not found, but DeleteObject is idempotent. Responding with 204 No Content.")
		return c.NoContent(http.StatusNoContent)
	}
	logger.Debug().Msg("Repository cloned successfully")

	err = h.gitasync.Delete(ctx, repo, bucketName, objectKey)
	if err != nil {
		if errors.Is(err, git.ErrFileNotExists) {
			logger.Info().Err(err).Msg("Object not found in repository, delete is idempotent.")
			// S3 returns 204 No Content if the object to be deleted is not found.
		} else {
			logger.Error().Err(err).Msg("Failed to delete object from repository")
			return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to delete object '%s': %v", objectKey, err))
		}
	} else {
		logger.Debug().Msg("Object deleted from repository successfully")
	}

	var deleteMarkerVersionID string
	headRef, headErr := repo.Head()
	if headErr != nil {
		logger.Warn().Err(headErr).Msg("Failed to get repository HEAD for versioning after delete.")
	} else if headRef != nil {
		deleteMarkerVersionID = headRef.Hash().String()
		logger.Debug().Str("deleteMarkerVersionID", deleteMarkerVersionID).Msg("Got commit SHA for delete marker version ID")
	} else {
		logger.Warn().Msg("repo.Head() returned nil ref without error after delete.")
	}

	c.Response().Header().Set("x-amz-delete-marker", "true")
	if deleteMarkerVersionID != "" {
		// For versioned buckets, S3 returns x-amz-version-id for the delete marker
		c.Response().Header().Set("x-amz-version-id", deleteMarkerVersionID)
	}

	logger.Info().Str("key", objectKey).Str("bucket", bucketName).Str("versionId", deleteMarkerVersionID).Msg("DeleteObject.OK")
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) CopyObjectAsync(c echo.Context) error {
	ctx := c.Request().Context()

	logger := log.Ctx(ctx).With().Str("command", "CopyObject").Bool("async", true).Logger()
	logger.Debug().Msg("CopyObject.Start")

	// Parse destination bucket and key
	destBucket := c.Param("bucket")
	if destBucket == "" {
		logger.Warn().Msg("Destination bucket name is missing")
		return c.String(http.StatusBadRequest, "Destination bucket name is missing")
	}

	destKey := c.Param("*")
	if destKey == "" {
		logger.Warn().Msg("Destination object key is missing")
		return c.String(http.StatusBadRequest, "Destination object key is missing")
	}
	destKey = strings.TrimPrefix(destKey, "/")

	// Parse source from x-amz-copy-source header
	copySource := c.Request().Header.Get("x-amz-copy-source")
	if copySource == "" {
		logger.Warn().Msg("x-amz-copy-source header is missing")
		return h.s3ErrorResponse(c, http.StatusBadRequest, "InvalidRequest", "x-amz-copy-source header is required for copy operations", destKey)
	}

	// Parse source bucket and key from copy source (format: /bucket/key)
	copySource = strings.TrimPrefix(copySource, "/")
	parts := strings.SplitN(copySource, "/", 2)
	if len(parts) != 2 {
		logger.Warn().Str("copy_source", copySource).Msg("Invalid x-amz-copy-source format")
		return h.s3ErrorResponse(c, http.StatusBadRequest, "InvalidRequest", "Invalid x-amz-copy-source format", destKey)
	}

	srcBucket := parts[0]
	srcKey := parts[1]

	logger = logger.With().
		Str("dest_bucket", destBucket).
		Str("dest_key", destKey).
		Str("src_bucket", srcBucket).
		Str("src_key", srcKey).
		Logger()

	logger.Debug().Msg("Parsed copy parameters")

	// For simplicity, this implementation only supports copying within the same bucket
	if srcBucket != destBucket {
		logger.Warn().Msg("Cross-bucket copying not supported")
		return h.s3ErrorResponse(c, http.StatusBadRequest, "InvalidRequest", "Cross-bucket copying is not supported", destKey)
	}

	// Clone the repository
	repo, err := h.gitasync.Clone(ctx, destBucket)
	if err != nil {
		logger.Error().Err(err).Str("bucket", destBucket).Msg("Failed to clone repository")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to clone repository '%s': %v", destBucket, err))
	}
	logger.Debug().Msg("Repository cloned successfully")

	// Perform the copy operation
	err = h.gitasync.Copy(ctx, repo, destBucket, srcKey, destKey)
	if err != nil {
		if errors.Is(err, git.ErrFileNotExists) {
			logger.Warn().Err(err).Str("src_key", srcKey).Msg("Source object not found")
			return h.s3ErrorResponse(c, http.StatusNotFound, "NoSuchKey", "The specified source key does not exist.", srcKey)
		}
		logger.Error().Err(err).Str("src_key", srcKey).Str("dest_key", destKey).Msg("Failed to copy object")
		return c.String(http.StatusInternalServerError, fmt.Sprintf("Failed to copy object: %v", err))
	}
	logger.Debug().Msg("Object copied successfully")

	// Get commit SHA for versioning
	var commitSHA string
	headRef, err := repo.Head()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get repository HEAD for versioning")
	} else {
		commitSHA = headRef.Hash().String()
		logger.Debug().Str("commitSHA", commitSHA).Msg("Got commit SHA for version ID")
	}

	// Set response headers
	c.Response().Header().Set("ETag", fmt.Sprintf(`"%s"`, commitSHA))
	if commitSHA != "" {
		c.Response().Header().Set("x-amz-version-id", commitSHA)
	}

	// Create and return CopyObjectResult XML response
	result := s3.CopyObjectResult{
		LastModified: time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		ETag:         fmt.Sprintf(`"%s"`, commitSHA),
	}

	logger.Info().Str("src_key", srcKey).Str("dest_key", destKey).Str("versionId", commitSHA).Msg("CopyObject.OK")
	return c.XML(http.StatusOK, result)
}
