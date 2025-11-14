package crypto

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"sync"
)

// Charset defines the character set to use for password generation
type Charset string

const (
	// CharsetAlpha uses only alphabetic characters (a-z, A-Z)
	CharsetAlpha Charset = "alpha"
	// CharsetAlnum uses alphanumeric characters (a-z, A-Z, 0-9)
	CharsetAlnum Charset = "alnum"
	// CharsetAlnumSym uses alphanumeric and special characters (a-z, A-Z, 0-9, !@#$%^&*()-_=+[]{}<>?,.:;/'\"|\\~)
	CharsetAlnumSym Charset = "alnumsym"
)

var (
	errInvalidLength   = errors.New("length must be positive")
	errUnknownCharset  = errors.New("unknown charset")
	errInvalidWordSize = errors.New("word count must be positive")
)

var (
	charsetLookup = map[Charset][]rune{
		CharsetAlpha:    []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"),
		CharsetAlnum:    []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"),
		CharsetAlnumSym: []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}<>?,.:;/'\"|\\~"),
	}
	randSource io.Reader = rand.Reader
	randMux    sync.RWMutex
)

var dicewareAdjectives = []string{
	"able", "amber", "brave", "calm", "clever", "crisp", "daring", "eager", "early", "fancy", "gentle", "happy", "ideal", "jolly", "keen", "lively", "magic", "noble", "oaken", "pearl", "quick", "ready", "solar", "tidy", "urban", "vivid", "warm", "young", "zesty", "bright", "candid", "dazzle", "elegant", "friendly", "glossy", "humble",
}

var dicewareNouns = []string{
	"anchor", "beacon", "canyon", "dream", "ember", "forest", "galaxy", "harbor", "island", "jungle", "kingdom", "lantern", "meadow", "nebula", "ocean", "prairie", "quartz", "river", "summit", "temple", "unicorn", "valley", "willow", "xenon", "yonder", "zephyr", "apple", "bridge", "comet", "dragon", "feather", "garden", "horizon", "idol", "jade", "keeper", "legend",
}

var (
	dicewareList []string
	dicewareOnce sync.Once
)

// SetRandomSource sets the random number generator source.
// If r is nil, it resets to the default crypto/rand.Reader.
func SetRandomSource(r io.Reader) {
	randMux.Lock()
	if r == nil {
		randSource = rand.Reader
	} else {
		randSource = r
	}
	randMux.Unlock()
}

// GeneratePassword generates a cryptographically secure random password with the specified length and character set.
// It returns the generated password or an error if the length is invalid or if there's a problem with the random source.
func GeneratePassword(length int, charset Charset) (string, error) {
	if length <= 0 {
		return "", errInvalidLength
	}

	chars, ok := charsetLookup[charset]
	if !ok {
		return "", errUnknownCharset
	}

	randMux.RLock()
	src := randSource
	randMux.RUnlock()

	var b strings.Builder
	b.Grow(length)

	for i := 0; i < length; i++ {
		idx, err := randomIndex(src, len(chars))
		if err != nil {
			return "", err
		}
		b.WriteRune(chars[idx])
	}

	return b.String(), nil
}

// GenerateDiceware generates a list of random words using the diceware method.
// wordCount specifies how many words to generate.
// Returns a slice of words or an error if wordCount is invalid or if there's a problem with the random source.
func GenerateDiceware(wordCount int) ([]string, error) {
	if wordCount <= 0 {
		return nil, errInvalidWordSize
	}

	words := dicewareWords()
	randMux.RLock()
	src := randSource
	randMux.RUnlock()

	result := make([]string, wordCount)
	for i := 0; i < wordCount; i++ {
		idx, err := randomIndex(src, len(words))
		if err != nil {
			return nil, err
		}
		result[i] = words[idx]
	}

	return result, nil
}

func dicewareWords() []string {
	dicewareOnce.Do(func() {
		pairs := len(dicewareAdjectives) * len(dicewareNouns)
		merged := make([]string, 0, pairs)
		for _, adj := range dicewareAdjectives {
			for _, noun := range dicewareNouns {
				merged = append(merged, adj+"-"+noun)
			}
		}
		dicewareList = merged
	})
	return dicewareList
}

func randomIndex(r io.Reader, max int) (int, error) {
	if max <= 0 {
		return 0, errInvalidLength
	}

	if max <= 256 {
		var buf [1]byte
		usable := 256 - (256 % max)
		for {
			if _, err := io.ReadFull(r, buf[:]); err != nil {
				return 0, err
			}
			if int(buf[0]) < usable {
				return int(buf[0]) % max, nil
			}
		}
	}

	if max <= 65536 {
		var buf [2]byte
		usable := 65536 - (65536 % max)
		for {
			if _, err := io.ReadFull(r, buf[:]); err != nil {
				return 0, err
			}
			val := int(binary.BigEndian.Uint16(buf[:]))
			if val < usable {
				return val % max, nil
			}
		}
	}

	var buf [4]byte
	const maxUint32 = ^uint32(0)
	limit := maxUint32 - (maxUint32 % uint32(max))
	for {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		val := binary.BigEndian.Uint32(buf[:])
		if val < limit {
			return int(val % uint32(max)), nil
		}
	}
}
