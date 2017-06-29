package constants

import (
	"errors"
	"fmt"
	"strings"
)

// Layout represents keyboard layout
type Layout struct {
	// Locales is a list of locales which could be used in pair with this layout
	Locales []string
	// Layout code
	Layout string
	// Description is a human readable description
	Description string
}

var layouts = []*Layout{
	&Layout{[]string{"be"}, "by", "Belarussian"},
	&Layout{[]string{"bg"}, "bg", "Bulgarian"},
	&Layout{[]string{"bs"}, "croat", "Bosnian"},
	&Layout{[]string{"cs", "cs_CZ"}, "cz-lat2", "Czech"},
	&Layout{[]string{"de_CH", "de_LI"}, "sg-latin1", "Swiss German"},
	&Layout{[]string{"de", "de_DE", "en_DE"}, "de-latin1-nodeadkeys", "German (Latin1; no dead keys)"},
	&Layout{[]string{"da"}, "dk-latin1", "Danish"},
	&Layout{[]string{"en", "en_CA", "en_US", "en_AU", "zh", "eo", "ko", "us", "nl", "nl_NL", "ar", "fa", "hi", "id", "mg", "ml", "gu", "pa", "kn", "dz", "ne", "sq", "tl", "vi", "xh"}, "us", "American"},
	&Layout{[]string{"en_IE", "en_GB", "en_GG", "en_IM", "en_JE", "ga", "gd", "gv", "cy", "kw"}, "uk", "British"},
	&Layout{[]string{"xx"}, "dvorak", "Dvorak"},
	&Layout{[]string{"et"}, "et", "Estonian"},
	&Layout{[]string{"ast", "ca", "es", "eu", "gl"}, "es", "Spanish"},
	&Layout{[]string{"es_CL", "es_DO", "es_GT", "es_HN", "es_MX", "es_PA", "es_PE", "es_SV"}, "la-latin1", "Latin American"},
	&Layout{[]string{"fi"}, "fi-latin1", "Finnish"},
	&Layout{[]string{"fr", "fr_FR", "br", "oc"}, "fr-latin9", "French "},
	&Layout{[]string{"fr_BE", "nl_BE", "wa"}, "be2-latin1", "Belgian"},
	&Layout{[]string{"fr_CA"}, "cf", "Canadian French"},
	&Layout{[]string{"fr_CH"}, "fr_CH-latin1", "Swiss French"},
	&Layout{[]string{"el"}, "gr", "Greek"},
	&Layout{[]string{"he"}, "hebrew", "Hebrew"},
	&Layout{[]string{"hr"}, "croat", "Croatian"},
	&Layout{[]string{"hu"}, "hu", "Hungarian"},
	&Layout{[]string{"is", "en_IS"}, "is-latin1", "Icelandic"},
	&Layout{[]string{"it"}, "it", "Italian"},
	&Layout{[]string{"ky"}, "ky", "Kirghiz"},
	&Layout{[]string{"lt"}, "lt", "Lithuanian"},
	&Layout{[]string{"lv"}, "lv-latin4", "Latvian"},
	&Layout{[]string{"ja", "ja_JP"}, "jp106", "Japanese (106 Key)"},
	&Layout{[]string{"mk"}, "mk", "Macedonian"},
	&Layout{[]string{"no", "nn", "nb", "se"}, "no-latin1", "Norwegian"},
	&Layout{[]string{"pl"}, "pl", "Polish"},
	&Layout{[]string{"pt"}, "pt-latin1", "Portuguese (Latin-1)"},
	&Layout{[]string{"pt_BR"}, "br-latin1", "Brazilian (Standard)"},
	&Layout{[]string{"pt_BR"}, "br-abnt2", "Brazilian (Standard ABNT2)"},
	&Layout{[]string{"ro"}, "ro", "Romanian"},
	&Layout{[]string{"ru"}, "ru", "Russian"},
	&Layout{[]string{"sk"}, "sk-qwerty", "Slovakian"},
	&Layout{[]string{"sl"}, "slovene", "Slovenian"},
	&Layout{[]string{"sr", "sr@latin"}, "sr", "Serbian"},
	&Layout{[]string{"sv"}, "se-latin1", "Swedish"},
	&Layout{[]string{"th"}, "th-tis", "Thai"},
	&Layout{[]string{"ku", "tr"}, "trfu", "Turkish (F layout)"},
	&Layout{[]string{"ku", "tr"}, "trqu", "Turkish (Q layout)"},
	&Layout{[]string{"uk"}, "ua", "Ukrainian"},
}

// LayoutList represents a slice of Layout with Strings method bound to it
type LayoutList []*Layout

// Strings returns slice of string descriptions
func (list LayoutList) Strings() []string {
	out := make([]string, len(list))
	for i, l := range list {
		out[i] = l.String()
	}
	return out
}

// GetLayout returns list of layouts for given layout and locale prefixes
func GetLayout(locale, layout string) LayoutList {
	var result LayoutList

	for _, l := range layouts {
		if !strings.HasPrefix(l.Layout, layout) {
			continue
		}

		for _, loc := range l.Locales {
			if strings.HasPrefix(loc, locale) {
				result = append(result, l)
				break
			}
		}
	}

	return result
}

// String returns human readable string representation for menus etc
func (l *Layout) String() string {
	return fmt.Sprintf("%s: %s", l.Layout, l.Description)
}

// ValidateLayout is a helper function for dialogs
func ValidateLayout(locale, layout string) error {
	list := GetLayout(locale, layout)
	if len(list) == 0 {
		return errors.New("No such layout")
	}
	return nil
}
