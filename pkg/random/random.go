package random

import (
	"math"
	"math/rand/v2"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/google/uuid"
)

const earthRadiusKM = 6371.0

// UUIDv1 generates a Version 1 UUID (timestamp + node ID).
func UUIDv1() string {
	u, err := uuid.NewUUID()
	if err != nil {
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
		return ""
	}
	return u.String()
}

// UUIDv7 generates a Version 7 UUID (Unix timestamp + random).
func UUIDv7() string {
	u, err := uuid.NewV7()
	if err != nil {
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

func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180
}

func radiansToDegrees(radians float64) float64 {
	return radians * 180 / math.Pi
}
