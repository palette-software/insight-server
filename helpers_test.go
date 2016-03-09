package insight_server

import (
	"testing"
)

func TestSanitizeName(t *testing.T) {
	assertString(t, SanitizeName("asd"), "asd", "Regular version escaped badly")
	assertString(t, SanitizeName("a_sd"), "a_sd", "Underscored version escaped badly")
	assertString(t, SanitizeName("a_s d"), "a_s-d", "Underscored-spaced version escaped badly")
	assertString(t, SanitizeName("a_s d*&:%"), "a_s-d----", "Underscored-spaced and misc characters version escaped badly")
}
