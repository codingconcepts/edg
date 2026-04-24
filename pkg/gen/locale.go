package gen

import (
	"fmt"
	"strings"

	"github.com/codingconcepts/edg/pkg/random"
)

var localeAliases = map[string]string{
	"en": "en_US", "us": "en_US",
	"ja": "ja_JP", "jp": "ja_JP",
	"de": "de_DE",
	"fr": "fr_FR",
	"es": "es_ES",
	"pt": "pt_BR", "br": "pt_BR",
	"zh": "zh_CN", "cn": "zh_CN",
	"ko": "ko_KR", "kr": "ko_KR",
}

func resolveLocale(locale string) (*localeData, error) {
	key := strings.ReplaceAll(locale, "-", "_")
	if ld, ok := locales[key]; ok {
		return ld, nil
	}
	lower := strings.ToLower(key)
	if alias, ok := localeAliases[lower]; ok {
		return locales[alias], nil
	}
	for k, ld := range locales {
		if strings.EqualFold(k, key) {
			return ld, nil
		}
	}
	return nil, fmt.Errorf("gen_locale: unknown locale %q", locale)
}

func pickRand(items []string) string {
	return items[random.Rng.IntN(len(items))]
}

func formatName(ld *localeData, first, last string) string {
	switch ld.NameOrder {
	case "eastern":
		return last + first
	default:
		return first + " " + last
	}
}

// GenLocale generates locale-aware PII data. Supported fields:
//
//	gen_locale('first_name', 'ja_JP')
//	gen_locale('name', 'de_DE')
//	gen_locale('city', 'fr_FR')
//	gen_locale('phone', 'ko_KR')
func GenLocale(field, locale string) (string, error) {
	ld, err := resolveLocale(locale)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(field) {
	case "first_name", "firstname":
		return pickRand(ld.FirstNames), nil
	case "last_name", "lastname":
		return pickRand(ld.LastNames), nil
	case "name", "full_name", "fullname":
		return formatName(ld, pickRand(ld.FirstNames), pickRand(ld.LastNames)), nil
	case "city":
		return pickRand(ld.Cities), nil
	case "street", "street_name", "streetname":
		return pickRand(ld.Streets), nil
	case "phone", "phone_number", "phonenumber":
		return random.Regex(ld.PhoneFmt), nil
	case "zip", "zip_code", "zipcode", "postal_code", "postalcode":
		return random.Regex(ld.ZipFmt), nil
	case "address":
		num := random.Rng.IntN(9999) + 1
		street := pickRand(ld.Streets)
		city := pickRand(ld.Cities)
		zip := random.Regex(ld.ZipFmt)
		switch ld.AddressFmt {
		case "eastern":
			return fmt.Sprintf("%s%s%d %s", city, street, num, zip), nil
		default:
			return fmt.Sprintf("%d %s, %s %s", num, street, city, zip), nil
		}
	default:
		return "", fmt.Errorf("gen_locale: unknown field %q (supported: first_name, last_name, name, city, street, phone, zip, address)", field)
	}
}

// SupportedLocales returns the list of supported locale codes.
func SupportedLocales() []string {
	keys := make([]string, 0, len(locales))
	for k := range locales {
		keys = append(keys, k)
	}
	return keys
}
