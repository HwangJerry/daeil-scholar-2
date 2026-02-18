// Summary transform — converts HTML content to plain-text summary
package main

import "github.com/dflh-saf/backend/internal/service"

func extractSummary(htmlContent string) string {
	plain := service.StripHTMLTags(htmlContent)
	return service.TruncateString(plain, 200)
}
