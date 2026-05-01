package domain

import "slices"

// ReservedMetadataKeys are the metadata fields that the billing layer
// owns on every Provider checkout. Providers MUST write these fields
// themselves; HTTP / API surfaces accepting caller-supplied metadata
// MUST drop them to prevent spoofing.
//
// Keep this list and Provider implementations in lockstep — the stripe
// provider asserts the set in its tests.
var ReservedMetadataKeys = []string{
	"user_id",
	"email",
	"plan",
	"interval",
	"product_type",
	"price_id",
	"quantity",
}

// IsReservedMetadataKey reports whether k is owned by the billing layer.
func IsReservedMetadataKey(k string) bool {
	return slices.Contains(ReservedMetadataKeys, k)
}
