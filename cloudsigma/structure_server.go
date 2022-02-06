package cloudsigma

import "github.com/cloudsigma/cloudsigma-sdk-go/cloudsigma"

func expandEnclavePageCaches(caches []interface{}) []cloudsigma.EnclavePageCache {
	expandedCaches := make([]cloudsigma.EnclavePageCache, 0, len(caches))

	for _, cache := range caches {
		c := &cloudsigma.EnclavePageCache{
			Size: cache.(int),
		}
		expandedCaches = append(expandedCaches, *c)
	}

	return expandedCaches
}

func flattenEnclavePageCaches(caches []cloudsigma.EnclavePageCache) []interface{} {
	flattenCaches := make([]interface{}, 0, len(caches))

	for _, cache := range caches {
		flattenCaches = append(flattenCaches, cache.Size)
	}

	return flattenCaches
}
