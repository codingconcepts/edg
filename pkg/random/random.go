package random

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"net"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

const earthRadiusKM = 6371.0

// UUIDv1 generates a Version 1 UUID (timestamp + node ID).
func UUIDv1() string {
	u, err := uuid.NewUUID()
	if err != nil {
		slog.Warn("uuid_v1: failed to generate", "error", err)
		return ""
	}
	return u.String()
}

// UUIDv4 generates a Version 4 UUID (random).
func UUIDv4() string {
	return uuid.NewString()
}

// UUIDv6 generates a Version 6 UUID (reordered timestamp).
func UUIDv6() string {
	u, err := uuid.NewV6()
	if err != nil {
		slog.Warn("uuid_v6: failed to generate", "error", err)
		return ""
	}
	return u.String()
}

// UUIDv7 generates a Version 7 UUID (Unix timestamp + random).
func UUIDv7() string {
	u, err := uuid.NewV7()
	if err != nil {
		slog.Warn("uuid_v7: failed to generate", "error", err)
		return ""
	}
	return u.String()
}

// Float generates a random float64 in [min, max] rounded to the given
// number of decimal places.
func Float(min, max float64, precision int) float64 {
	v := min + rand.Float64()*(max-min)
	shift := math.Pow(10, float64(precision))

	return math.Round(v*shift) / shift
}

// Uniform generates a uniform random float64 in [min, max].
func Uniform(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

// Zipf generates a Zipfian-distributed random integer in [0, imax].
// Parameters s (> 1) and v (>= 1) control the distribution shape.
// Returns 0 if s <= 1 or v < 1.
func Zipf(s, v float64, imax int) int {
	src := rand.NewPCG(rand.Uint64(), rand.Uint64())
	r := rand.New(src)
	z := rand.NewZipf(r, s, v, uint64(imax))
	if z == nil {
		return 0
	}
	return int(z.Uint64())
}

// Norm generates a normally-distributed random float64 clamped to
// [min, max]. If precision is provided, the result is rounded to that
// many decimal places; otherwise it is rounded to 0 (whole number).
func Norm(mean, stddev, min, max float64, precision ...int) float64 {
	p := 0
	if len(precision) > 0 {
		p = precision[0]
	}
	shift := math.Pow(10, float64(p))

	for {
		v := mean + stddev*rand.NormFloat64()
		rounded := math.Round(v*shift) / shift
		if rounded >= min && rounded <= max {
			return rounded
		}
	}
}

// Exp generates an exponentially-distributed random float64 with the given
// rate (lambda). The result is clamped to [min, max] and rounded to the
// specified number of decimal places (default 0).
func Exp(rate, min, max float64, precision ...int) float64 {
	p := 0
	if len(precision) > 0 {
		p = precision[0]
	}
	shift := math.Pow(10, float64(p))

	for {
		v := rand.ExpFloat64() / rate
		rounded := math.Round(v*shift) / shift
		if rounded >= min && rounded <= max {
			return rounded
		}
	}
}

// LogNorm generates a log-normally-distributed random float64 clamped to
// [min, max]. mu and sigma are the mean and standard deviation of the
// underlying normal distribution. The result is rounded to the specified
// number of decimal places (default 0).
func LogNorm(mu, sigma, min, max float64, precision ...int) float64 {
	p := 0
	if len(precision) > 0 {
		p = precision[0]
	}
	shift := math.Pow(10, float64(p))

	for {
		v := math.Exp(mu + sigma*rand.NormFloat64())
		rounded := math.Round(v*shift) / shift
		if rounded >= min && rounded <= max {
			return rounded
		}
	}
}

// Regex generates a random string matching the given regular expression.
func Regex(pattern string) string {
	return gofakeit.Regex(pattern)
}

// Point generates a random geographic point within radiusKM of (lat, lon).
func Point(lat, lon, radiusKM float64) (float64, float64) {
	randomDistance := (rand.Float64() * radiusKM) / earthRadiusKM
	randomBearing := rand.Float64() * 2 * math.Pi

	latRad := degreesToRadians(lat)
	lonRad := degreesToRadians(lon)

	sinLatRad := math.Sin(latRad)
	cosLatRad := math.Cos(latRad)
	sinRandomDistance := math.Sin(randomDistance)
	cosRandomDistance := math.Cos(randomDistance)
	cosRandomBearing := math.Cos(randomBearing)
	sinRandomBearing := math.Sin(randomBearing)

	newLatRad := math.Asin(sinLatRad*cosRandomDistance + cosLatRad*sinRandomDistance*cosRandomBearing)

	newLonRad := lonRad + math.Atan2(
		sinRandomBearing*sinRandomDistance*cosLatRad,
		cosRandomDistance-sinLatRad*math.Sin(newLatRad),
	)

	return radiansToDegrees(newLatRad), radiansToDegrees(newLonRad)
}

// Timestamp generates a random time.Time between min and max.
func Timestamp(min, max time.Time) time.Time {
	if min.Equal(max) {
		return min
	}

	if min.After(max) {
		min, max = max, min
	}

	minUnix := min.Unix()
	maxUnix := max.Unix()
	delta := maxUnix - minUnix

	randUnix := minUnix + rand.Int64N(delta)
	return time.Unix(randUnix, 0)
}

// Duration generates a random time.Duration between min and max.
func Duration(min, max time.Duration) time.Duration {
	if min == max {
		return min
	}

	if min > max {
		min, max = max, min
	}

	diff := max - min
	randomDiff := time.Duration(rand.Int64N(int64(diff)))

	return min + randomDiff
}

// Bytes generates n random bytes and returns them as a hex-encoded
// string prefixed with \x, matching CockroachDB/PostgreSQL BYTES literal format.
func Bytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.IntN(256))
	}
	return `\x` + hex.EncodeToString(b)
}

// Bit generates a random fixed-length bit string of exactly n bits.
func Bit(n int) string {
	b := make([]byte, n)
	for i := range n {
		b[i] = '0' + byte(rand.IntN(2))
	}
	return string(b)
}

// VarBit generates a random variable-length bit string of 1 to n bits.
func VarBit(n int) string {
	if n <= 0 {
		return ""
	}
	return Bit(1 + rand.IntN(n))
}

// Inet generates a random IP address within the given CIDR block.
// Supports both IPv4 and IPv6.
func Inet(cidr string) (string, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return "", fmt.Errorf("inet: invalid CIDR %q: %w", cidr, err)
	}

	ip := network.IP
	mask := network.Mask

	result := make(net.IP, len(ip))
	for i := range ip {
		result[i] = (ip[i] & mask[i]) | (byte(rand.IntN(256)) & ^mask[i])
	}

	return result.String(), nil
}

// TimeOfDay generates a random time of day between min and max,
// both in HH:MM:SS format.
func TimeOfDay(minStr, maxStr string) (string, error) {
	minSec, err := parseTimeOfDay(minStr)
	if err != nil {
		return "", fmt.Errorf("time: invalid min %q: %w", minStr, err)
	}
	maxSec, err := parseTimeOfDay(maxStr)
	if err != nil {
		return "", fmt.Errorf("time: invalid max %q: %w", maxStr, err)
	}

	if minSec > maxSec {
		minSec, maxSec = maxSec, minSec
	}

	randSec := minSec
	if maxSec > minSec {
		randSec += rand.IntN(maxSec - minSec + 1)
	}

	h := randSec / 3600
	m := (randSec % 3600) / 60
	s := randSec % 60

	return fmt.Sprintf("%02d:%02d:%02d", h, m, s), nil
}

func parseTimeOfDay(s string) (int, error) {
	t, err := time.Parse("15:04:05", s)
	if err != nil {
		return 0, err
	}
	return t.Hour()*3600 + t.Minute()*60 + t.Second(), nil
}

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}
