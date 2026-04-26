package stripe

func getString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getInt64(m map[string]any, key string) int64 {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int64(n)
		case int64:
			return n
		case int:
			return int64(n)
		}
	}
	return 0
}

func getBool(m map[string]any, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getMap(m map[string]any, key string) map[string]any {
	if v, ok := m[key]; ok {
		if sub, ok := v.(map[string]any); ok {
			return sub
		}
	}
	return map[string]any{}
}

func getSlice(m map[string]any, keys ...string) []any {
	current := m
	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := current[key]; ok {
				if arr, ok := v.([]any); ok {
					return arr
				}
			}
			return nil
		}
		if sub, ok := current[key].(map[string]any); ok {
			current = sub
		} else {
			return nil
		}
	}
	return nil
}
