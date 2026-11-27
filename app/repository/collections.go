package repository

// Nama koleksi MongoDB. Disimpan di package repository supaya repository
// tidak perlu mengimpor package database (mencegah import cycle
// repository -> database -> config -> route -> repository).
const (
	CollectionModuleContents = "module_contents"
	CollectionProgress       = "progress"
	CollectionReviews        = "module_reviews"
)
