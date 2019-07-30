package httpclient

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/version"
)

func TestUserAgentString_env(t *testing.T) {
	expectedBase := fmt.Sprintf(userAgentFormat, version.Version)
	if oldenv, isSet := os.LookupEnv(uaEnvVar); isSet {
		defer os.Setenv(uaEnvVar, oldenv)
	} else {
		defer os.Unsetenv(uaEnvVar)
	}

	for i, c := range []struct {
		expected   string
		additional string
	}{
		{expectedBase, ""},
		{expectedBase, " "},
		{expectedBase, " \n"},

		{fmt.Sprintf("%s test/1", expectedBase), "test/1"},
		{fmt.Sprintf("%s test/2", expectedBase), "test/2 "},
		{fmt.Sprintf("%s test/3", expectedBase), " test/3 "},
		{fmt.Sprintf("%s test/4", expectedBase), "test/4 \n"},
	} {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if c.additional == "" {
				os.Unsetenv(uaEnvVar)
			} else {
				os.Setenv(uaEnvVar, c.additional)
			}

			actual := UserAgentString()

			if c.expected != actual {
				t.Fatalf("Expected User-Agent '%s' does not match '%s'", c.expected, actual)
			}
		})
	}
}

func TestUserAgentProductEqual(t *testing.T) {
	first := &UserAgentProduct{"ProductName", "version", ""}
	second := &UserAgentProduct{"ProductName", "version", ""}

	if !first.Equal(second) {
		t.Fatalf("Unexpected mismatch between %q and %q", first, second)
	}

	different := &UserAgentProduct{"ProductName", "different", ""}

	if first.Equal(different) {
		t.Fatalf("Unexpected match between %q and %q", first, different)
	}

	withComment := &UserAgentProduct{"ProductName", "different", "comm"}
	if first.Equal(withComment) {
		t.Fatalf("Unexpected match between %q and %q", first, withComment)
	}
}

func TestParseUserAgentString(t *testing.T) {
	testCases := []struct {
		uaString          string
		useragentProducts []*UserAgentProduct
		expectError       bool
	}{
		{
			"terraform-github-actions/1.0",
			[]*UserAgentProduct{{"terraform-github-actions", "1.0", ""}},
			false,
		},
		{
			"TFE/a718e58f",
			[]*UserAgentProduct{{"TFE", "a718e58f", ""}},
			false,
		},
		{
			"OneProduct/0.1.0 AnotherOne/1.2",
			[]*UserAgentProduct{{"OneProduct", "0.1.0", ""}, {"AnotherOne", "1.2", ""}},
			false,
		},
		{
			"ProductWithComment/1.0.0 (a comment; goes; here)",
			[]*UserAgentProduct{{"ProductWithComment", "1.0.0", "a comment; goes; here"}},
			false,
		},
		{
			"ProductWithComment/1.0.0 (a comment; goes; here) AnotherProductWithComment/5.5.0 (blah)",
			[]*UserAgentProduct{
				{"ProductWithComment", "1.0.0", "a comment; goes; here"},
				{"AnotherProductWithComment", "5.5.0", "blah"},
			},
			false,
		},
		{
			"NoComment/1.0.0 AnotherProductWithComment/5.5.0 (blah)",
			[]*UserAgentProduct{
				{"NoComment", "1.0.0", ""},
				{"AnotherProductWithComment", "5.5.0", "blah"},
			},
			false,
		},
		{
			"First/1.0.0 Second/5.5.0 Third/5.5.0",
			[]*UserAgentProduct{
				{"First", "1.0.0", ""},
				{"Second", "5.5.0", ""},
				{"Third", "5.5.0", ""},
			},
			false,
		},
		{
			"MissingVersion",
			[]*UserAgentProduct{},
			true,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			parsedUA, err := ParseUserAgentString(tc.uaString)
			if err != nil {
				if tc.expectError {
					return
				}
				t.Fatal(err)
			}

			givenUA := newUserAgent(parsedUA)
			expectedUA := newUserAgent(tc.useragentProducts)

			if !givenUA.Equal(expectedUA) {
				t.Fatalf("Unexpected User-Agent.\nExpected: %q\nGiven: %q\n", expectedUA, givenUA)
			}
		})
	}

}

func TestUserAgentAppendViaEnvVar(t *testing.T) {
	if oldenv, isSet := os.LookupEnv(uaEnvVar); isSet {
		defer os.Setenv(uaEnvVar, oldenv)
	} else {
		defer os.Unsetenv(uaEnvVar)
	}

	testCases := []struct {
		baseUserAgent []*UserAgentProduct
		envVarValue   string
		expected      []*UserAgentProduct
	}{
		{
			[]*UserAgentProduct{},
			"",
			[]*UserAgentProduct{},
		},
		{
			[]*UserAgentProduct{},
			" ",
			[]*UserAgentProduct{},
		},
		{
			[]*UserAgentProduct{},
			" \n",
			[]*UserAgentProduct{},
		},
		{
			[]*UserAgentProduct{{"Foo", "1.0", ""}},
			"test/1",
			[]*UserAgentProduct{{"Foo", "1.0", ""}, {"test", "1", ""}},
		},
		{
			[]*UserAgentProduct{},
			"test/1 (comment)",
			[]*UserAgentProduct{{"test", "1", "comment"}},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			os.Unsetenv(uaEnvVar)
			expectedUA := newUserAgent(tc.expected)

			os.Setenv(uaEnvVar, tc.envVarValue)
			givenUA := newUserAgent(tc.baseUserAgent)
			if !givenUA.Equal(expectedUA) {
				t.Fatalf("Expected User-Agent '%s' does not match '%s'", expectedUA, givenUA)
			}
		})
	}
}

func TestUserAgentString(t *testing.T) {
	simpleUA := newUserAgent([]*UserAgentProduct{
		{"First", "1.0", ""},
	})
	expectedSimpleUA := "First/1.0"
	givenSimpleUA := simpleUA.String()
	if givenSimpleUA != expectedSimpleUA {
		t.Fatalf("Expected UA string to be %q, given %q", expectedSimpleUA, givenSimpleUA)
	}

	withComment := newUserAgent([]*UserAgentProduct{
		{"Bar", "2.5.1", "random comment; foo"},
	})
	expectedWithComment := "Bar/2.5.1 (random comment; foo)"
	givenWithComment := withComment.String()
	if givenWithComment != expectedWithComment {
		t.Fatalf("Expected UA string to be %q, given %q", expectedWithComment, givenWithComment)
	}

	multiProducts := newUserAgent([]*UserAgentProduct{
		{"Foo", "v1", ""},
		{"Bar", "2.5.1", "random comment; foo"},
	})
	expectedMultiProducts := "Foo/v1 Bar/2.5.1 (random comment; foo)"
	givenMultiProducts := multiProducts.String()
	if givenMultiProducts != expectedMultiProducts {
		t.Fatalf("Expected UA string to be %q, given %q", expectedMultiProducts, givenMultiProducts)
	}
}

func TestUserAgentAppend(t *testing.T) {
	ua := newUserAgent([]*UserAgentProduct{
		{"Foo", "1.0.0", ""},
	})
	givenUA := ua.Append(&UserAgentProduct{"Bar", "2.0", "boo"})

	expectedUA := newUserAgent([]*UserAgentProduct{
		{"Foo", "1.0.0", ""},
		{"Bar", "2.0", "boo"},
	})

	if !givenUA.Equal(expectedUA) {
		t.Fatalf("Unexpected mismatch between the following:\n%q\n%q", givenUA, expectedUA)
	}
}
