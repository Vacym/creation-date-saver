package jsonstore

import "creation-date-saver/domain"

func domainMetadataToModel(meta domain.Metadata) Metadata {
	return Metadata{
		CreationTime: meta.CreationTime,
	}
}

func modelMetadataToDomain(path string, meta Metadata) domain.Metadata {
	return domain.Metadata{
		Patch:        path,
		CreationTime: meta.CreationTime,
	}
}
