package shape

import (
	"go/ast"
	"strings"

	"github.com/widmogrod/mkunion/x/shared"
)

// ExtractDocumentTags extracts struct tags from Go comment groups.
// It uses the unified ExtractTags function which now handles generic type
// parameters in any tag, not just mkunion tags.
func ExtractDocumentTags(doc *ast.CommentGroup) map[string]Tag {
	result := make(map[string]Tag)

	comments := strings.Split(shared.Comment(doc), "\n")
	for _, comment := range comments {
		if strings.HasPrefix(comment, "go:tag") {
			tagString := strings.TrimPrefix(comment, "go:tag")
			tagString = strings.TrimSpace(tagString)
			for k, v := range ExtractTags(tagString) {
				result[k] = v
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// ExtractTags parses Go struct tags with support for generic type parameters.
// Unlike the standard structtag.Parse, this function correctly handles brackets
// with commas inside them, making it suitable for any tag that uses generic syntax.
//
// The key improvement is that commas inside brackets are treated as part of the
// tag value, not as option separators. This allows tags to contain generic types
// like Map[K, V] or Either[A, B] without breaking the parsing.
//
// Examples:
//   - sometag:"Option[A]"
//   - othertag:"Map[K, V],option1,option2"
//   - mkunion:"Either[A, B],serde"
//   - anytag:"Complex[Nested[A, B], C]"
//
// Special handling:
// - Empty tags return nil
// - Malformed tags are skipped silently
func ExtractTags(tag string) map[string]Tag {
	tag = strings.Trim(tag, "`")
	tag = strings.TrimSpace(tag)

	// Handle special case for tags without values
	if tag != "" && !strings.Contains(tag, ":") && !strings.Contains(tag, ",") {
		return map[string]Tag{
			tag: {
				Value:   "",
				Options: nil,
			},
		}
	}

	if tag == "" {
		return nil
	}

	result := make(map[string]Tag)

	// Parse tags manually with custom logic to handle brackets correctly.
	// This replaces structtag.Parse to support generic type parameters.
	remaining := tag
	for remaining != "" {
		remaining = strings.TrimSpace(remaining)

		// Find the next tag key
		colonIdx := strings.Index(remaining, ":")
		if colonIdx == -1 {
			break
		}

		// Handle potential space before the tag key
		spaceIdx := strings.LastIndex(remaining[:colonIdx], " ")
		startIdx := 0
		if spaceIdx != -1 {
			// Check if this space is inside quotes
			if !isSpaceInsideQuotes(remaining[:colonIdx]) {
				startIdx = spaceIdx + 1
			}
		}

		key := strings.TrimSpace(remaining[startIdx:colonIdx])
		remaining = remaining[colonIdx+1:]

		// Check if the value starts with a quote
		if !strings.HasPrefix(remaining, "\"") {
			// Malformed tag, skip
			continue
		}

		// Extract the tag value with bracket-aware parsing
		tag, nextRemaining := extractTagWithBrackets(key, remaining[1:]) // Skip opening quote
		if tag.Value != "" || len(tag.Options) > 0 {
			result[key] = tag
		}

		remaining = nextRemaining
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

// extractTagWithBrackets extracts a tag value and options with bracket-aware parsing.
// It handles commas inside brackets as part of the value, not as option separators.
//
// For example, in the tag `mkunion:"Either[A, B],serde,omitempty"`:
//   - value: "Either[A, B]" (comma inside brackets is preserved)
//   - options: ["serde", "omitempty"] (commas outside brackets are separators)
//
// Parameters:
//   - key: The tag key (unused but kept for potential future use)
//   - valueString: The tag content after the opening quote
//
// Returns:
//   - Tag: The parsed tag with value and options
//   - string: Remaining unparsed string
func extractTagWithBrackets(key, valueString string) (Tag, string) {
	// Find the end of the tag value, accounting for brackets
	endIdx := findTagValueEndUnified(valueString, 0)
	if endIdx == -1 {
		return Tag{}, ""
	}

	value := valueString[:endIdx]
	var options []string
	remaining := ""

	// Check what comes after the value
	if endIdx < len(valueString) {
		switch valueString[endIdx] {
		case '"':
			// End of tag value, check for more tags
			if endIdx+1 < len(valueString) {
				remaining = valueString[endIdx+1:]
			}
		case ',':
			// We have options
			optEnd := strings.Index(valueString[endIdx:], "\"")
			if optEnd != -1 {
				optStr := valueString[endIdx+1 : endIdx+optEnd]
				if optStr != "" {
					opts := strings.Split(optStr, ",")
					for _, opt := range opts {
						opt = strings.TrimSpace(opt)
						if opt != "" {
							options = append(options, opt)
						}
					}
				}
				if endIdx+optEnd+1 < len(valueString) {
					remaining = valueString[endIdx+optEnd+1:]
				}
			}
		}
	}

	return Tag{
		Value:   value,
		Options: options,
	}, remaining
}

// findTagValueEndUnified finds the end index of a tag value, accounting for brackets.
// It tracks bracket depth to ensure commas inside generic type parameters
// are not mistaken for tag option separators.
//
// The function returns:
//   - The index of the first quote (end of value)
//   - The index of the first comma outside of brackets (if followed by options)
//   - The string length if no separator is found (and brackets are balanced)
//   - -1 if brackets are mismatched
func findTagValueEndUnified(s string, startIdx int) int {
	bracketDepth := 0

	for i := startIdx; i < len(s); i++ {
		switch s[i] {
		case '[':
			bracketDepth++
		case ']':
			bracketDepth--
			if bracketDepth < 0 {
				// Mismatched brackets
				return -1
			}
		case '"':
			// Quote ends the tag value
			return i
		case ',':
			// Only treat comma as separator if we're outside brackets
			if bracketDepth == 0 {
				return i
			}
		}
	}

	// If we reach here and brackets are balanced, return length
	if bracketDepth == 0 {
		return len(s)
	}

	return -1
}

// isSpaceInsideQuotes checks if there's an unclosed quote in the string,
// which would mean any space found is inside quotes
func isSpaceInsideQuotes(s string) bool {
	inQuotes := false
	for _, c := range s {
		if c == '"' {
			inQuotes = !inQuotes
		}
	}
	return inQuotes
}

// parseUnionNameWithTypeParams parses a union name that may contain type parameters.
// This is used specifically for parsing mkunion tag values to extract the base name
// and type parameter list.
//
// Examples:
//   - "Tree" returns ("Tree", nil)
//   - "Tree[T]" returns ("Tree", ["T"])
//   - "Result[T, E]" returns ("Result", ["T", "E"])
//   - "Map[K, V]" returns ("Map", ["K", "V"])
func parseUnionNameWithTypeParams(name string) (string, []string) {
	bracketIdx := strings.Index(name, "[")
	if bracketIdx == -1 {
		return name, nil
	}

	baseName := name[:bracketIdx]

	if !strings.HasSuffix(name, "]") {
		return name, nil
	}

	paramsStr := name[bracketIdx+1 : len(name)-1]
	if paramsStr == "" {
		return baseName, []string{}
	}

	params := strings.Split(paramsStr, ",")
	for i := range params {
		params[i] = strings.TrimSpace(params[i])
	}

	return baseName, params
}

// formatTypeParamsForTag formats type parameters for display in tags or error messages.
// Examples:
//   - [] returns ""
//   - ["T"] returns "[T]"
//   - ["K", "V"] returns "[K, V]"
func formatTypeParamsForTag(params []TypeParam) string {
	if len(params) == 0 {
		return ""
	}

	names := make([]string, len(params))
	for i, param := range params {
		names[i] = param.Name
	}

	return "[" + strings.Join(names, ", ") + "]"
}

func TagsToGuard(tags map[string]Tag) Guard {
	var result Guard
	if enum, ok := tags["enum"]; ok {
		result = ConcatGuard(result, &Enum{
			Val: append(strings.Split(enum.Value, ","), enum.Options...),
		})
	}
	if required, ok := tags["required"]; ok && required.Value == "true" {
		result = ConcatGuard(result, &Required{})
	}

	return result
}

func TagsToDesc(tags map[string]Tag) *string {
	if desc, ok := tags["desc"]; ok {
		// because tags are parsed according to the spec, we need to normalize options
		// since description field does not support options
		value := strings.Join(append([]string{desc.Value}, desc.Options...), ", ")
		descStr := strings.Trim(value, `"`)
		if descStr != "" {
			return &descStr
		}
	}

	return nil
}

// MergeTagsInto merges newTags into tags, this is mutable operation
func MergeTagsInto(tags map[string]Tag, newTags map[string]Tag) map[string]Tag {
	if tags == nil {
		return newTags
	}

	for k, v := range newTags {
		tags[k] = v
	}

	return tags
}
