package shape

import (
	"github.com/fatih/structtag"
	"github.com/widmogrod/mkunion/x/shared"
	"go/ast"
	"strings"
)

func ExtractDocumentTags(doc *ast.CommentGroup) map[string]Tag {
	result := make(map[string]Tag)

	comments := strings.Split(shared.Comment(doc), "\n")
	for _, comment := range comments {
		if strings.HasPrefix(comment, "go:tag") {
			tagString := strings.TrimPrefix(comment, "go:tag")
			tags := ExtractTags(tagString)
			for k, v := range tags {
				result[k] = v
			}
		}
	}

	if len(result) == 0 {
		return nil
	}

	return result
}

func ExtractTags(tag string) map[string]Tag {
	tag = strings.Trim(tag, "`")
	tags, err := structtag.Parse(tag)
	if err != nil {
		return nil
	}

	if len(tags.Tags()) == 0 {
		return nil
	}

	result := make(map[string]Tag)
	for _, t := range tags.Tags() {
		result[t.Key] = Tag{
			Value:   t.Name,
			Options: t.Options,
		}
	}

	return result
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
		value := strings.Join(append([]string{desc.Value}, desc.Options...), ",")
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
