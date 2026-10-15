package operations

import "encoding/json"

func stringJSON(value any) string { raw, _ := json.Marshal(value); return string(raw) }
