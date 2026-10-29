package checks

import "strings"

const minSharedTerms = 2

var stopwords = func() map[string]bool {
	set := map[string]bool{}
	for w := range strings.FieldsSeq("a about all an and are as at be been by can for from has have in into is it its of on or our that the their this to we will with you your who which than then them they was were what when where while using use able also not but more most such") {
		set[w] = true
	}
	return set
}()

func ClosestSentence(bullet, text string) (string, bool) {
	want := wordSet(bullet)
	for w := range want {
		if stopwords[w] {
			delete(want, w)
		}
	}
	best, bestShared := "", 0
	for _, s := range sentenceRe.Split(text, -1) {
		have := wordSet(s)
		shared := 0
		for w := range want {
			if have[w] {
				shared++
			}
		}
		if shared > bestShared {
			best, bestShared = strings.TrimSpace(s), shared
		}
	}
	if bestShared < minSharedTerms {
		return "", false
	}
	return best, true
}
