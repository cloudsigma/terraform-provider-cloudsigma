package cloudsigma

import "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"

func expandTags(tags []interface{}) []cloudsigma.Tag {
	expandedTags := make([]cloudsigma.Tag, 0, len(tags))

	for _, tag := range tags {
		t := &cloudsigma.Tag{
			UUID: tag.(string),
		}
		expandedTags = append(expandedTags, *t)
	}

	return expandedTags
}

func flattenTags(tags []cloudsigma.Tag) []interface{} {
	flattenTags := make([]interface{}, 0, len(tags))

	for _, tag := range tags {
		flattenTags = append(flattenTags, tag.UUID)
	}

	return flattenTags
}
