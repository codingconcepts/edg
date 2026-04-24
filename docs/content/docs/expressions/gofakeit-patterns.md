---
title: gofakeit Patterns
weight: 4
---

# Available gofakeit Patterns

These patterns can be used with `gen()`, `gen_batch()`, `json_arr()`, and `array()`. Patterns are case-insensitive. Parameters are separated from the function name by `:` and from each other by `,`.

```yaml
# No parameters
- gen('email')

# With parameters
- gen('number:1,100')

# In a batch
- gen_batch(1000, 100, 'email')

# In a JSON array
- json_arr(1, 5, 'firstname')

# In a PostgreSQL/CockroachDB array
- array(2, 4, 'word')
```

Patterns are validated at config load time. A typo like `gen('emial')` produces a clear error instead of silently returning the literal string `{emial}`.

## Address

| Pattern | Description | Example |
|---|---|---|
| `address` | Full address (street, city, state, zip, country) | `{street: 37802 Port Streetborough, city: Chesapeake, state: North Carolina, zip: 18508, country: Andorra}` |
| `city` | City name | `Mardarville` |
| `country` | Country name | `United States` |
| `countryabr` | 2-letter country code | `US` |
| `latitude` | Latitude coordinate | `41.7886` |
| `latituderange:0,90` | Latitude in range (default 0–90) | `52.31` |
| `longitude` | Longitude coordinate | `-112.0591` |
| `longituderange:0,180` | Longitude in range (default 0–180) | `73.45` |
| `state` | State name | `Idaho` |
| `stateabr` | 2-letter state abbreviation | `ID` |
| `street` | Full street (number + name + suffix) | `364 East Parkway` |
| `streetname` | Street name | `View` |
| `streetnumber` | Street number | `364` |
| `streetprefix` | Directional prefix (N, E, SW) | `East` |
| `streetsuffix` | Street type (Ave, St, Blvd) | `Parkway` |
| `timezone` | Timezone name | `America/New_York` |
| `timezoneabv` | 3-letter timezone abbreviation | `EST` |
| `timezonefull` | Full timezone name | `Eastern Standard Time` |
| `timezoneoffset` | UTC offset | `-5` |
| `timezoneregion` | Timezone region | `America` |
| `unit` | Building unit (apt, suite) | `Apt 204` |
| `zip` | Postal code | `83201` |

## Airline

| Pattern | Description | Example |
|---|---|---|
| `airlineaircrafttype` | Aircraft category | `Narrow-body` |
| `airlineairplane` | Aircraft model | `Boeing 737` |
| `airlineairport` | Airport name | `Heathrow Airport` |
| `airlineairportiata` | IATA airport code | `LHR` |
| `airlineflightnumber` | Flight number | `BA142` |
| `airlinerecordlocator` | Booking reference | `XJDF42` |
| `airlineseat` | Seat assignment | `14A` |

## Animals

| Pattern | Description | Example |
|---|---|---|
| `animal` | Animal name | `Lion` |
| `animaltype` | Animal type (mammal, bird, etc.) | `mammal` |
| `bird` | Bird species | `Eagle` |
| `cat` | Cat breed | `Siamese` |
| `dog` | Dog breed | `Labrador` |
| `farmanimal` | Farm animal | `Cow` |
| `petname` | Pet name | `Buddy` |

## Color

| Pattern | Description | Example |
|---|---|---|
| `color` | Color name | `MediumSlateBlue` |
| `hexcolor` | Hex color code | `#1a2b3c` |
| `nicecolors` | Curated color palette | `[#e8d5b7, #0e2430, ...]` |
| `rgbcolor` | RGB color values | `[52, 152, 219]` |
| `safecolor` | Web-safe color name | `fuchsia` |

## Company

| Pattern | Description | Example |
|---|---|---|
| `blurb` | Company description | `We provide scalable...` |
| `bs` | Business buzzword phrase | `synergize scalable mindshare` |
| `buzzword` | Business buzzword | `synergize` |
| `company` | Company name | `Acme Corp` |
| `companysuffix` | Company suffix (Inc., LLC) | `Inc.` |
| `job` | Job details | `{company: Google, title: Contractor, descriptor: District, level: Assurance}` |
| `jobdescriptor` | Job descriptor | `Senior` |
| `joblevel` | Job level | `Manager` |
| `jobtitle` | Job title | `Software Engineer` |
| `slogan` | Company slogan | `Think different.` |

## Contact

| Pattern | Description | Example |
|---|---|---|
| `email` | Email address | `markusmoen@pagac.net` |
| `phone` | Phone number | `6136459211` |
| `phoneformatted` | Formatted phone number | `(613) 645-9211` |
| `username` | Account username | `markus.moen` |

## Data Structures

| Pattern | Description | Example |
|---|---|---|
| `csv:,,10` | CSV rows (delimiter, row count) | `name,email\nAlice,...` |
| `fixed_width:10` | Fixed-width format | `Alice     ...` |
| `json:object,10` | JSON document (type, field count) | `{"name": "..."}` |
| `map` | Random key-value map | `{interest: 5418, only: 2991258, fly: {shall: 1188343}}` |
| `sql:,10` | SQL INSERT statements | `INSERT INTO ...` |
| `svg:500,500` | SVG image | `<svg>...</svg>` |
| `template:` | Template-driven text | *(from template)* |
| `xml:single,xml,record,10` | XML document | `<record>...</record>` |

## Date & Time

| Pattern | Description | Example |
|---|---|---|
| `date:RFC3339` | Date in specified format | `2023-07-15T14:32:07Z` |
| `daterange:2020-01-01,2025-12-31,yyyy-MM-dd` | Date in range with format | `2023-07-15` |
| `day` | Day of month | `15` |
| `futuredate` | Date in the future | `2027-03-21T10:00:00Z` |
| `hour` | Hour (0–23) | `14` |
| `minute` | Minute (0–59) | `32` |
| `month` | Month number (1–12) | `7` |
| `monthstring` | Month name | `July` |
| `nanosecond` | Nanosecond | `196519854` |
| `pastdate` | Date in the past | `2019-11-05T08:30:00Z` |
| `second` | Second (0–59) | `7` |
| `time:HH:mm:ss` | Time in format (default HH:mm:ss) | `14:32:07` |
| `timerange:08:00:00,17:00:00,HH:mm:ss` | Time in range with format | `12:45:23` |
| `weekday` | Weekday name | `Wednesday` |
| `year` | Year | `2023` |

## Emoji

| Pattern | Description | Example |
|---|---|---|
| `emoji` | Random emoji | `🎉` |
| `emojialias` | Emoji alias keyword | `:tada:` |
| `emojianimal` | Animal emoji | `🐕` |
| `emojicategory` | Emoji category | `Smileys & Emotion` |
| `emojiclothing` | Clothing emoji | `👗` |
| `emojicostume` | Costume/fantasy emoji | `🧛` |
| `emojielectronics` | Electronics emoji | `📱` |
| `emojiface` | Face/smiley emoji | `😊` |
| `emojiflag` | Flag emoji | `🇺🇸` |
| `emojifood` | Food emoji | `🍕` |
| `emojigame` | Game emoji | `🎮` |
| `emojigesture` | Gesture emoji | `🤷` |
| `emojihand` | Hand emoji | `👍` |
| `emojijob` | Job/role emoji | `👨‍🔬` |
| `emojilandmark` | Landmark emoji | `🗽` |
| `emojimusic` | Music emoji | `🎸` |
| `emojiperson` | Person emoji | `👩` |
| `emojiplant` | Plant emoji | `🌻` |
| `emojisentence:3` | Sentence with emojis (N emojis) | `I am 😊 and 🎉 today 🌟` |
| `emojisport` | Sport emoji | `⚽` |
| `emojitag` | Emoji tag | `happy` |
| `emojitools` | Tools emoji | `🔧` |
| `emojivehicle` | Vehicle emoji | `🚗` |
| `emojiweather` | Weather emoji | `☀️` |

## Entertainment

| Pattern | Description | Example |
|---|---|---|
| `book` | Book details | `{title: Sons and Lovers, author: James Joyce, genre: Saga}` |
| `bookauthor` | Book author | `F. Scott Fitzgerald` |
| `bookgenre` | Book genre | `Fiction` |
| `booktitle` | Book title | `The Great Gatsby` |
| `celebrityactor` | Celebrity actor | `Tom Hanks` |
| `celebritybusiness` | Business celebrity | `Elon Musk` |
| `celebritysport` | Sports celebrity | `Serena Williams` |
| `gamertag` | Gaming username | `xX_Slayer_Xx` |
| `hobby` | Hobby or pastime | `Photography` |
| `movie` | Movie details | `{name: Sherlock Jr., genre: Music}` |
| `moviegenre` | Movie genre | `Sci-Fi` |
| `moviename` | Movie title | `The Matrix` |
| `song` | Song details | `{name: Agora Hills, artist: Olivia Newton-John, genre: Country}` |
| `songartist` | Song artist | `Queen` |
| `songgenre` | Song genre | `Rock` |
| `songname` | Song title | `Bohemian Rhapsody` |

## Error Messages

| Pattern | Description | Example |
|---|---|---|
| `error` | Error message | `unexpected EOF` |
| `errordatabase` | Database error | `connection refused` |
| `errorgrpc` | gRPC error | `deadline exceeded` |
| `errorhttp` | HTTP error | `404 Not Found` |
| `errorhttpclient` | HTTP client error | `timeout awaiting...` |
| `errorhttpserver` | HTTP server error | `502 Bad Gateway` |
| `errorobject` | Error object | `{code: 500, message: service unavailable}` |
| `errorruntime` | Runtime error | `index out of bounds` |
| `errorvalidation` | Validation error | `field required` |

## Finance

| Pattern | Description | Example |
|---|---|---|
| `achaccount` | ACH bank account number | `586981958265` |
| `achrouting` | 9-digit ACH routing number | `071000013` |
| `bankname` | Bank name | `Chase` |
| `banktype` | Bank type | `Commercial` |
| `bitcoinaddress` | Bitcoin address | `1A1zP1eP5QGefi2D...` |
| `bitcoinprivatekey` | Bitcoin private key | `5HueCGU8rMjxEXx...` |
| `creditcard` | Full credit card details | `{type: UnionPay, number: 6376121963702920, exp: 10/29, cvv: 505}` |
| `creditcardcvv` | Credit card CVV | `513` |
| `creditcardexp` | Credit card expiry | `02/28` |
| `creditcardnumber` | Credit card number (default: any type) | `4111111111111111` |
| `creditcardtype` | Credit card type | `Visa` |
| `currency` | Currency details | `{short: ZAR, long: South Africa Rand}` |
| `currencylong` | Full currency name | `United States Dollar` |
| `currencyshort` | 3-letter currency code | `USD` |
| `cusip` | CUSIP security identifier | `38259P508` |
| `ein` | Employer Identification Number | `12-3456789` |
| `isin` | ISIN security identifier | `US38259P5081` |
| `price:0,1000` | Price in range (default 0–1000) | `42.99` |

## Food & Drink

| Pattern | Description | Example |
|---|---|---|
| `beeralcohol` | Beer alcohol content | `5.2%` |
| `beerblg` | Beer gravity (BLG) | `12.5` |
| `beerhop` | Beer hop variety | `Cascade` |
| `beeribu` | Beer bitterness (IBU) | `40` |
| `beermalt` | Beer malt type | `Pale Ale` |
| `beername` | Beer name | `Duvel` |
| `beerstyle` | Beer style | `IPA` |
| `beeryeast` | Beer yeast strain | `Safale US-05` |
| `breakfast` | Breakfast food | `Scrambled eggs` |
| `dessert` | Dessert item | `Chocolate cake` |
| `dinner` | Dinner food | `Grilled salmon` |
| `drink` | Drink | `Lemonade` |
| `fruit` | Fruit | `Apple` |
| `lunch` | Lunch food | `Caesar salad` |
| `snack` | Snack item | `Trail mix` |
| `vegetable` | Vegetable | `Broccoli` |

## Grammar

| Pattern | Description | Example |
|---|---|---|
| `adjective` | General adjective | `bright` |
| `adjectivedemonstrative` | Demonstrative adjective (this, that) | `this` |
| `adjectivedescriptive` | Descriptive adjective | `beautiful` |
| `adjectiveindefinite` | Indefinite adjective | `some` |
| `adjectiveinterrogative` | Interrogative adjective | `which` |
| `adjectivepossessive` | Possessive adjective | `our` |
| `adjectiveproper` | Proper adjective | `American` |
| `adjectivequantitative` | Quantitative adjective | `several` |
| `adverb` | General adverb | `quickly` |
| `adverbdegree` | Degree adverb | `very` |
| `adverbfrequencydefinite` | Definite frequency adverb | `daily` |
| `adverbfrequencyindefinite` | Indefinite frequency adverb | `often` |
| `adverbmanner` | Manner adverb | `carefully` |
| `adverbplace` | Place adverb | `here` |
| `adverbtimedefinite` | Definite time adverb | `yesterday` |
| `adverbtimeindefinite` | Indefinite time adverb | `soon` |
| `connective` | Connective word | `however` |
| `connectivecasual` | Causal connective | `because` |
| `connectivecomparative` | Comparative connective | `similarly` |
| `connectivecomplaint` | Complaint connective | `unfortunately` |
| `connectiveexamplify` | Example connective | `for instance` |
| `connectivelisting` | Listing connective | `firstly` |
| `connectivetime` | Time connective | `meanwhile` |
| `interjection` | Interjection | `wow` |
| `noun` | General noun | `table` |
| `nounabstract` | Abstract noun | `freedom` |
| `nouncollectiveanimal` | Animal collective noun | `flock` |
| `nouncollectivepeople` | People collective noun | `crowd` |
| `nouncollectivething` | Thing collective noun | `bundle` |
| `nouncommon` | Common noun | `book` |
| `nounconcrete` | Concrete noun | `chair` |
| `nouncountable` | Countable noun | `apple` |
| `noundeterminer` | Noun determiner | `the` |
| `nounproper` | Proper noun | `London` |
| `noununcountable` | Uncountable noun | `water` |
| `preposition` | General preposition | `with` |
| `prepositioncompound` | Compound preposition | `in front of` |
| `prepositiondouble` | Double preposition | `out of` |
| `prepositionsimple` | Simple preposition | `at` |
| `pronoun` | General pronoun | `she` |
| `pronoundemonstrative` | Demonstrative pronoun | `these` |
| `pronounindefinite` | Indefinite pronoun | `someone` |
| `pronouninterrogative` | Interrogative pronoun | `who` |
| `pronounobject` | Object pronoun | `him` |
| `pronounpersonal` | Personal pronoun | `I` |
| `pronounpossessive` | Possessive pronoun | `mine` |
| `pronounreflective` | Reflexive pronoun | `myself` |
| `pronounrelative` | Relative pronoun | `which` |
| `verb` | General verb | `run` |
| `verbaction` | Action verb | `jump` |
| `verbhelping` | Helping verb | `would` |
| `verbintransitive` | Intransitive verb | `sleep` |
| `verblinking` | Linking verb | `seem` |
| `verbtransitive` | Transitive verb | `carry` |

## Hacker

| Pattern | Description | Example |
|---|---|---|
| `hackerabbreviation` | Hacker abbreviation | `SQL` |
| `hackeradjective` | Hacker adjective | `back-end` |
| `hackeringverb` | Hacker -ing verb | `hacking` |
| `hackernoun` | Hacker noun | `firewall` |
| `hackerphrase` | Hacker phrase | `Use the neural TCP...` |
| `hackerverb` | Hacker verb | `parse` |

## Internet

| Pattern | Description | Example |
|---|---|---|
| `apiuseragent` | API client user agent | `curl/7.68.0` |
| `chromeuseragent` | Chrome user agent | `Mozilla/5.0 ... Chrome/...` |
| `domainname` | Domain name | `example.com` |
| `domainsuffix` | Domain suffix | `.com` |
| `firefoxuseragent` | Firefox user agent | `Mozilla/5.0 ... Firefox/...` |
| `httpmethod` | HTTP method | `GET` |
| `httpstatuscode` | HTTP status code | `404` |
| `httpstatuscodesimple` | Common HTTP status code | `200` |
| `httpversion` | HTTP version | `1.1` |
| `inputname` | HTML input element name | `first_name` |
| `ipv4address` | IPv4 address | `192.168.1.42` |
| `ipv6address` | IPv6 address | `2001:db8::1` |
| `macaddress` | MAC address | `00:1A:2B:3C:4D:5E` |
| `operauseragent` | Opera user agent | `Mozilla/5.0 ... OPR/...` |
| `safariuseragent` | Safari user agent | `Mozilla/5.0 ... Safari/...` |
| `url` | Web URL | `https://www.example.com/path` |
| `urlslug:3` | URL-safe slug (N words, default 3) | `modern-web-design` |
| `useragent` | Browser user agent string | `Mozilla/5.0 ...` |

## Language

| Pattern | Description | Example |
|---|---|---|
| `language` | Language name | `English` |
| `languageabbreviation` | Language abbreviation | `en` |
| `languagebcp` | BCP 47 language tag | `en-US` |
| `programminglanguage` | Programming language | `Go` |

## Minecraft

| Pattern | Description | Example |
|---|---|---|
| `minecraftanimal` | Minecraft animal | `Cow` |
| `minecraftarmorpart` | Armor piece | `Chestplate` |
| `minecraftarmortier` | Armor tier | `Diamond` |
| `minecraftbiome` | Minecraft biome | `Plains` |
| `minecraftdye` | Minecraft dye color | `Red` |
| `minecraftfood` | Minecraft food | `Bread` |
| `minecraftmobboss` | Boss mob | `Ender Dragon` |
| `minecraftmobhostile` | Hostile mob | `Creeper` |
| `minecraftmobneutral` | Neutral mob | `Enderman` |
| `minecraftmobpassive` | Passive mob | `Sheep` |
| `minecraftore` | Minecraft ore | `Diamond` |
| `minecrafttool` | Minecraft tool | `Pickaxe` |
| `minecraftvillagerjob` | Villager job | `Librarian` |
| `minecraftvillagerlevel` | Villager level | `Journeyman` |
| `minecraftvillagerstation` | Villager station | `Lectern` |
| `minecraftweapon` | Minecraft weapon | `Sword` |
| `minecraftweather` | Minecraft weather | `Rain` |
| `minecraftwood` | Minecraft wood type | `Oak` |

## Miscellaneous

| Pattern | Description | Example |
|---|---|---|
| `email_text` | Email message body | `Dear Sir/Madam...` |
| `imagejpeg:500,500` | Random JPEG image (W×H) | *(binary image data)* |
| `imagepng:500,500` | Random PNG image (W×H) | *(binary image data)* |
| `loglevel` | Log severity level | `error` |
| `password:true,true,true,true,false,12` | Password (lower, upper, numeric, special, space, length) | `aB3$kL9mPq2x` |
| `randomint:` | Random pick from int array | *(selected int)* |
| `randomstring:` | Random pick from string array | *(selected string)* |
| `randomuint:` | Random pick from uint array | *(selected uint)* |
| `shuffleints:` | Shuffle int array | *(shuffled array)* |
| `shufflestrings:` | Shuffle string array | *(shuffled array)* |
| `teams:,` | Split people into teams | *(team assignments)* |
| `weighted:,` | Weighted random selection | *(selected value)* |

## Numbers

| Pattern | Description | Example |
|---|---|---|
| `bool` | true or false | `true` |
| `dice:2,[6,6]` | Dice roll (count, sides per die) | `[4, 2]` |
| `digit` | Single digit (0–9) | `7` |
| `digitn:6` | String of N digits | `482910` |
| `flipacoin` | Coin toss | `Heads` |
| `float32` | 32-bit float | `3.14` |
| `float32range:1,10` | 32-bit float in range | `7.23` |
| `float64` | 64-bit float | `3.141592653` |
| `float64range:0,1` | 64-bit float in range | `0.7312` |
| `hexuint:8` | Hex unsigned integer (N hex chars) | `4a3f2b1c` |
| `int` | Random signed integer | `8294723` |
| `int8` | Signed 8-bit integer (−128 to 127) | `42` |
| `int16` | Signed 16-bit integer | `8294` |
| `int32` | Signed 32-bit integer | `829472389` |
| `int64` | Signed 64-bit integer | `8294723891234` |
| `intn:100` | Integer in [0, N) | `73` |
| `intrange:1,100` | Signed integer in range | `67` |
| `number:1,100` | Integer in range (default full int32 range) | `42` |
| `uint` | Unsigned integer | `4294967` |
| `uint8` | Unsigned 8-bit integer (0–255) | `200` |
| `uint16` | Unsigned 16-bit integer (0–65535) | `50000` |
| `uint32` | Unsigned 32-bit integer | `2948293` |
| `uint64` | Unsigned 64-bit integer | `394857239482` |
| `uintn:100` | Unsigned integer in [0, N) | `42` |
| `uintrange:0,1000` | Unsigned integer in range | `512` |

## Person

| Pattern | Description | Example |
|---|---|---|
| `age` | Age in years | `32` |
| `bio` | Random biography | `I'm a developer from NY...` |
| `ethnicity` | Cultural or ethnic background | `Caucasian` |
| `firstname` | Given name | `Markus` |
| `gender` | Gender classification | `male` |
| `lastname` | Family name | `Moen` |
| `middlename` | Middle name | `James` |
| `name` | Full name (first and last) | `Markus Moen` |
| `nameprefix` | Title or honorific (Mr., Mrs., Dr.) | `Mr.` |
| `namesuffix` | Suffix (Jr., Sr., III) | `Jr.` |
| `person` | Full personal details (name, contact, etc.) | `{first_name: Jessica, last_name: Hills, gender: female, age: 51, ssn: 961445393, ...}` |
| `ssn` | US Social Security Number | `296-28-1925` |

## Product

| Pattern | Description | Example |
|---|---|---|
| `product` | Product details | `{name: Water Dispenser, price: 91.59, material: cardboard, upc: 058601249007, ...}` |
| `productaudience` | Target audience | `Professionals` |
| `productbenefit` | Key product benefit | `Time-saving` |
| `productcategory` | Product category | `Electronics` |
| `productdescription` | Product description | `High-quality wireless...` |
| `productdimension` | Product dimensions | `10x5x3 inches` |
| `productfeature` | Product feature | `Waterproof` |
| `productisbn` | ISBN identifier | `978-3-16-148410-0` |
| `productmaterial` | Product material | `Stainless Steel` |
| `productname` | Product name | `Ergonomic Keyboard` |
| `productsuffix` | Product model suffix | `Pro` |
| `productupc` | UPC barcode | `012345678905` |
| `productusecase` | Product use case | `Office productivity` |

## School

| Pattern | Description | Example |
|---|---|---|
| `school` | School name | `Lincoln High School` |

## Social Media

| Pattern | Description | Example |
|---|---|---|
| `socialmedia` | Social media handle/URL | `@johndoe` |

## String Manipulation

| Pattern | Description | Example |
|---|---|---|
| `generate:{firstname} {lastname}` | Generate from template | `Alice Smith` |
| `id` | Short URL-safe base32 identifier | `01hzxq5v8k` |
| `lexify:???` | Replace `?` with random letters | `kqb` |
| `numerify:###` | Replace `#` with random digits | `482` |
| `regex:[A-Z]{3}-[0-9]{4}` | String matching regex | `ABK-7291` |
| `uuid` | RFC 4122 v4 UUID | `550e8400-e29b-41d4-a716-446655440000` |

## Text

| Pattern | Description | Example |
|---|---|---|
| `comment` | Comment or remark | `This is great work!` |
| `hipsterparagraph:2,5,1,\n` | Hipster paragraph | *(multi-sentence hipster text)* |
| `hipstersentence:5` | Hipster sentence (N words) | `Artisan cold-pressed vegan...` |
| `hipsterword` | Hipster vocabulary word | `artisan` |
| `letter` | Single ASCII letter | `g` |
| `lettern:8` | String of N letters | `abcqwzml` |
| `loremipsumparagraph:2,5,1,\n` | Lorem Ipsum paragraph | *(multi-sentence Lorem Ipsum)* |
| `loremipsumsentence:5` | Lorem Ipsum sentence (N words) | `Lorem ipsum dolor sit amet.` |
| `loremipsumword` | Lorem Ipsum word | `lorem` |
| `markdown` | Markdown-formatted text | *(formatted markdown)* |
| `paragraph:3,5,12,\n` | Paragraph (sentences, words, paragraphs, separator) | *(multi-sentence text)* |
| `phrase` | Short phrase | `a quiet afternoon` |
| `phraseadverb` | Adverb phrase | `very carefully` |
| `phrasenoun` | Noun phrase | `the old house` |
| `phrasepreposition` | Prepositional phrase | `in the garden` |
| `phraseverb` | Verb phrase | `runs quickly` |
| `question` | Question sentence | `Where did you go?` |
| `quote` | Quoted text | `"To be or not to be"` |
| `sentence:5` | Sentence with N words (default 5) | `The quick brown fox jumped.` |
| `vowel` | Single vowel | `e` |
| `word` | Random word | `themselves` |

## Vehicle

| Pattern | Description | Example |
|---|---|---|
| `car` | Car details | `{type: Passenger car heavy, fuel: Ethanol, transmission: Automatic, brand: Alfa Romeo, model: Lancer, year: 2014}` |
| `carfueltype` | Fuel type | `Electric` |
| `carmaker` | Car manufacturer | `Toyota` |
| `carmodel` | Car model | `Camry` |
| `cartransmissiontype` | Transmission type | `Automatic` |
| `cartype` | Car type | `Sedan` |
