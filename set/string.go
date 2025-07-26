package set

import "strings"

// FromStr converts a delimited string into a Set of strings.
func FromStr(str, sep string) Set[string] {
	result := New[string]()
	items := strings.Split(str, sep)
	for _, raw := range items {
		item := strings.Trim(raw, " ,;.'\"?/><\\|}]{[_-=+`~!@#$%^&*()")
		if item != "" {
			result.Add(item)
		}
	}
	return result
}

// ToStr joins the items of a Set into a single string separated by sep.
func ToStr(set Set[string], sep string) string {
	strList := make([]string, 0, len(set))
	for val := range set {
		strList = append(strList, val)
	}
	return strings.Join(strList, sep)
}
