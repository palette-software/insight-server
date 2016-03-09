package insight_server

import (
    "testing"
    "fmt"
)

func TestSanitizeName(t *testing.T) {
    expected := "asd"
    actual := SanitizeName("asd")
    assert(t, actual == expected, fmt.Sprintf("Expected: %s while got: %s", expected, actual))
    expected = "a_sd"
    actual = SanitizeName("a_sd")
    assert(t, actual == expected, fmt.Sprintf("Expected: %s while got: %s", expected, actual))
}
