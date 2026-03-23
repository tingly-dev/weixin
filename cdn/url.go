// Package cdn provides CDN utilities for WeChat media upload/download.
package cdn

import (
	"net/url"
)

// BuildDownloadURL builds a CDN download URL from encrypted query param.
func BuildDownloadURL(encryptedQueryParam, cdnBaseURL string) string {
	return cdnBaseURL + "/download?encrypted_query_param=" + url.QueryEscape(encryptedQueryParam)
}

// BuildUploadURL builds a CDN upload URL from upload param and filekey.
func BuildUploadURL(cdnBaseURL, uploadParam, filekey string) string {
	return cdnBaseURL + "/upload?encrypted_query_param=" + url.QueryEscape(uploadParam) + "&filekey=" + url.QueryEscape(filekey)
}
