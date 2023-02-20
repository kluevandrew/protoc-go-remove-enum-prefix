package main

func isTaggedEnum(comment string) bool {
	return rComment.MatchString(comment)
}
func replaceEnumValueIdent(contents []byte, area EnumValueIdentArea) []byte {
	var injected []byte
	injected = append(injected, contents[:area.Start-1]...)
	injected = append(injected, area.NewName...)
	injected = append(injected, contents[area.End-1:]...)

	return injected
}

func removeEnumComment(contents []byte, area CommentArea) []byte {
	var injected []byte

	start := area.Start - 1

	prevChar := string(contents[area.Start-2])
	if prevChar == "\n" {
		start--
	}

	injected = append(injected, contents[:start]...)
	injected = append(injected, contents[area.End-1:]...)

	return injected
}
