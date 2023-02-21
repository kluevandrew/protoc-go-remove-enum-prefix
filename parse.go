package main

func isTaggedEnum(comment string) bool {
	return rComment.MatchString(comment)
}
