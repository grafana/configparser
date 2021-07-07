package configparser

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type receivedSection struct {
	name    string
	options map[string][2]string // 0: full value, 1: s.ValueOfWithoutComments()
}

func TestBasic(t *testing.T) {
	type testCase struct {
		title     string
		in        string
		expErr    bool
		expGlobal receivedSection
		expOther  []receivedSection
	}

	testCases := []testCase{
		{
			title:  "completely empty",
			in:     "",
			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: nil,
		},
		{
			title:  "empty name ", // note: this is legal. but up for debate i suppose
			in:     `[]`,
			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name:    "",
					options: make(map[string][2]string),
				},
			},
		},
		{
			title: "bad name format",
			in: `foo[]
						`,
			expErr: true,
		},
		{
			title:  "section without opts",
			in:     `[foo]`,
			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name:    "foo",
					options: make(map[string][2]string),
				},
			},
		},
		{
			title: "section with lots of whitespace and without opts", // the newline causes the "" option to be set.
			in: `              [foo]
					`,
			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name: "foo",
					options: map[string][2]string{
						"": [2]string{"", ""},
					},
				},
			},
		},
		{
			title:  "strange section syntax", // note: this is legal!
			in:     `              [[[foo[][]`,
			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name:    "foo[",
					options: make(map[string][2]string),
				},
			},
		},
		{
			title: "section with opts and lots of comments", // pretty weird, everything is tracked as-is. something up for debate. https://github.com/alyu/configparser/issues/11
			in: `   [foo] # comment here
				foo = bar # another comment
                                

				# [bar]

# [baz] # commented out twice
# also commented out
				`,

			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name: "foo",
					options: map[string][2]string{
						"foo":                           {"bar # another comment", "bar"},
						"":                              {"", ""},
						"# also commented out":          {"", ""},
						"# [bar]":                       {"", ""}, // why is a commented out section name marked as an option of a prior section?
						"# [baz] # commented out twice": {"", ""},
					},
				},
			},
		},
		{
			title: "values and comments can legally contain special chars. values can contain any char but # marks a comment",
			in: `   [foo] # comment here
				opt1 = ^[^;]+\.max(?:;|$)
				opt2 = bar # ^[^;]+\.max(?:;|$)
				opt3 # ^[^;]+\.max(?:;|$)
				opt4 = a;b:c=d
				opt5 = foo#bar#baz
				opt6 = :;[](){}!@$%^&*=+-_\|/?<>.,"'~#
				opt7 = :;[](){}!@$%^&*=+-_\|/?<>.,"'~#ab
				`,

			expErr: false,
			expGlobal: receivedSection{
				options: make(map[string][2]string),
			},
			expOther: []receivedSection{
				receivedSection{
					name: "foo",
					options: map[string][2]string{
						"opt1":                      {`^[^;]+\.max(?:;|$)`, `^[^;]+\.max(?:;|$)`},
						"opt2":                      {`bar # ^[^;]+\.max(?:;|$)`, `bar`},
						`opt3 # ^[^;]+\.max(?:;|$)`: {"", ""},
						"opt4":                      {"a;b:c=d", "a;b:c=d"},
						"opt5":                      {"foo#bar#baz", "foo"},
						"opt6":                      {`:;[](){}!@$%^&*=+-_\|/?<>.,"'~#`, `:;[](){}!@$%^&*=+-_\|/?<>.,"'~`},
						"opt7":                      {`:;[](){}!@$%^&*=+-_\|/?<>.,"'~#ab`, `:;[](){}!@$%^&*=+-_\|/?<>.,"'~`},
						"":                          {"", ""},
					},
				},
			},
		},
	}

	for _, c := range testCases {
		t.Logf("testing %q", c.title)
		conf, err := Read(strings.NewReader(c.in), "/tmp/configparser-test")
		if c.expErr && err == nil {
			t.Fatalf("testcase %q expected error but got no error", c.title)
		}
		if !c.expErr && err != nil {
			t.Fatalf("testcase %q expected no error but got error %s", c.title, err.Error())
		}
		if !c.expErr {
			global, other, _ := conf.AllSections()
			gotGlobal := convertSection(global)
			gotOther := convertSections(other)
			if !reflect.DeepEqual(c.expGlobal, gotGlobal) {
				t.Fatalf("testcase %q mismatch\nexp global section %+v\ngot global section %+v", c.title, c.expGlobal, gotGlobal)
			}
			if !reflect.DeepEqual(c.expOther, gotOther) {
				fmt.Println("exp other")
				fmt.Println("got other")
				t.Fatalf("testcase %q mismatch\nexp sections %+v\ngot sections %+v", c.title, c.expOther, gotOther)
			}
		}
	}
}

func convertSection(s *Section) receivedSection {
	options := s.Options()
	receivedOptions := make(map[string][2]string)
	for k, v := range options {
		receivedOptions[k] = [2]string{v, s.ValueOfWithoutComments(k)}
	}
	return receivedSection{
		name:    s.Name(),
		options: receivedOptions,
	}
}

func convertSections(ss []*Section) []receivedSection {
	var out []receivedSection
	for _, s := range ss {
		out = append(out, convertSection(s))
	}
	return out
}
