package storage

// DefaultEmailTemplateSeedsForExport returns a copy of the V1 seeds.
// Exposed so the api package can reset templates to defaults.
func DefaultEmailTemplateSeedsForExport() []*EmailTemplate {
	return defaultEmailTemplateSeeds()
}
