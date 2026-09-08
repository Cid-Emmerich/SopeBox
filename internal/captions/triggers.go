// Package captions turns transcript words into on-screen captions and
// picks the animated emoji that a spoken word deserves.
package captions

import (
	"strings"
	"unicode"

	"github.com/Cid-Emmerich/SopeBox/internal/glyph"
)

// extra maps words (beyond the glyphs' own names and aliases) to a glyph
// name. Only words that are fun and not so common that the panel never
// rests are included.
var extra = map[string]string{
	// drinks
	"coffee": "coffee", "espresso": "coffee", "latte": "coffee", "caffeine": "coffee",
	"tea": "tea", "matcha": "tea",
	"beer": "beer", "beers": "beer", "pint": "beer", "brewery": "beer", "drunk": "beer",
	"noodles": "ramen", "ramen": "ramen", "soup": "ramen", "dinner": "ramen", "lunch": "ramen", "breakfast": "coffee",
	// feelings
	"love": "heart", "loved": "heart", "loving": "heart", "heart": "heart", "hearts": "heart", "valentine": "heart",
	"laugh": "joy", "laughing": "joy", "laughed": "joy", "hilarious": "joy", "funny": "joy", "lol": "joy", "haha": "joy", "hahaha": "joy", "laughter": "joy", "laughs": "joy", "joke": "joy", "jokes": "joy",
	"smile": "grin", "smiling": "grin", "happy": "grin", "happiness": "grin", "glad": "grin",
	"wink": "wink", "flirt": "wink", "flirting": "wink",
	"cool": "cool", "sunglasses": "cool", "chill": "cool",
	"sleep": "sleeping", "sleepy": "sleeping", "asleep": "sleeping", "tired": "sleeping", "exhausted": "sleeping", "nap": "sleeping", "snoring": "sleeping",
	"wow": "mind blown", "whoa": "mind blown", "insane": "mind blown", "unbelievable": "mind blown", "mindblowing": "mind blown", "incredible": "mind blown", "wild": "mind blown",
	"think": "thinking", "thinking": "thinking", "hmm": "thinking", "wonder": "thinking", "wondering": "thinking", "ponder": "thinking", "philosophy": "thinking",
	"party": "party", "celebrate": "party", "celebration": "party", "congratulations": "party", "congrats": "party", "confetti": "party", "cheers": "party",
	"scared": "scream", "scary": "scream", "terrifying": "scream", "terrified": "scream", "horror": "scream", "nightmare": "scream", "scream": "scream", "screaming": "scream",
	"robot": "robot", "robots": "robot", "android": "robot", "ai": "robot", "chatgpt": "robot", "algorithm": "robot", "automation": "robot",
	"angry": "angry", "mad": "angry", "furious": "angry", "rage": "angry", "pissed": "angry", "hate": "angry", "hated": "angry",
	"upside": "upside down", "silly": "upside down", "weird": "upside down", "ironic": "upside down", "sarcastic": "upside down",
	// animals
	"cat": "cat", "cats": "cat", "kitten": "cat", "kitty": "cat", "meow": "cat",
	"dog": "dog", "dogs": "dog", "puppy": "dog", "puppies": "dog", "woof": "dog",
	"ghost": "ghost", "ghosts": "ghost", "haunted": "ghost", "spooky": "ghost", "halloween": "ghost",
	"dead": "skull", "death": "skull", "died": "skull", "skull": "skull", "skeleton": "skull", "die": "skull", "killed": "skull", "murder": "skull",
	"snake": "snake", "snakes": "snake", "python": "snake", "serpent": "snake",
	"butterfly": "butterfly", "butterflies": "butterfly", "moth": "butterfly",
	"fish": "fish", "fishing": "fish", "salmon": "fish", "shark": "fish", "aquarium": "fish",
	"bee": "bee", "bees": "bee", "honey": "bee", "wasp": "bee", "buzzing": "bee",
	"frog": "frog", "frogs": "frog", "toad": "frog",
	"turtle": "turtle", "tortoise": "turtle", "slow": "turtle", "slowly": "turtle",
	"crab": "crab", "crabs": "crab", "lobster": "crab", "rust": "crab",
	"octopus": "octopus", "squid": "octopus", "tentacles": "octopus", "kraken": "octopus",
	"owl": "owl", "owls": "owl", "wise": "owl", "wisdom": "owl",
	// people
	"hello": "wave", "hi": "wave", "hey": "wave", "bye": "wave", "goodbye": "wave", "welcome": "wave", "greetings": "wave",
	"run": "runner", "running": "runner", "ran": "runner", "marathon": "runner", "sprint": "runner", "jog": "runner", "jogging": "runner",
	"dance": "dancer", "dancing": "dancer", "danced": "dancer", "salsa": "dancer", "disco": "dancer",
	"yoga": "meditate", "meditate": "meditate", "meditation": "meditate", "zen": "meditate", "calm": "meditate", "breathe": "meditate", "mindfulness": "meditate",
	"cartwheel": "cartwheel", "gymnastics": "cartwheel", "flip": "cartwheel", "acrobat": "cartwheel",
	"gym": "lifter", "weights": "lifter", "workout": "lifter", "lifting": "lifter", "deadlift": "lifter", "squat": "lifter", "bench": "lifter", "muscle": "lifter", "muscles": "lifter",
	"clap": "clap", "clapping": "clap", "applause": "clap", "bravo": "clap",
	"question": "raise", "questions": "raise", "volunteer": "raise", "ask": "raise",
	"walk": "walker", "walking": "walker", "walked": "walker", "stroll": "walker", "hike": "walker", "hiking": "walker",
	// objects
	"rocket": "rocket", "rockets": "rocket", "launch": "rocket", "space": "rocket", "nasa": "rocket", "spacex": "rocket", "mars": "rocket", "astronaut": "rocket", "orbit": "rocket",
	"clock": "clock", "alarm": "clock", "o'clock": "clock", "deadline": "clock", "schedule": "clock",
	"wait": "hourglass", "waiting": "hourglass", "patience": "hourglass", "patient": "hourglass", "hourglass": "hourglass",
	"snowman": "snowman", "christmas": "snowman",
	"balloon": "balloon", "balloons": "balloon", "float": "balloon", "floating": "balloon",
	"idea": "bulb", "ideas": "bulb", "lightbulb": "bulb", "genius": "bulb", "insight": "bulb", "brilliant": "bulb", "eureka": "bulb",
	"bomb": "bomb", "explode": "bomb", "exploded": "bomb", "explosion": "bomb", "boom": "bomb", "kaboom": "bomb", "blew": "bomb", "dynamite": "bomb",
	"music": "music", "song": "music", "songs": "music", "sing": "music", "singing": "music", "melody": "music", "guitar": "music", "piano": "music", "album": "music", "concert": "music", "band": "music",
	"target": "target", "goal": "target", "goals": "target", "bullseye": "target", "aim": "target", "focus": "target", "darts": "target",
	"dice": "dice", "gamble": "dice", "gambling": "dice", "random": "dice", "casino": "dice", "vegas": "dice", "bet": "dice", "lucky": "dice", "luck": "dice",
	"gift": "gift", "gifts": "gift", "present": "gift", "presents": "gift", "surprise": "gift",
	"battery": "battery", "batteries": "battery", "charge": "battery", "charging": "battery", "charged": "battery",
	"cake": "cake", "birthday": "cake", "dessert": "cake", "cupcake": "cake",
	"refresh": "refresh", "reload": "refresh", "loading": "refresh", "update": "refresh", "updated": "refresh", "sync": "refresh",
	"signal": "signal", "wifi": "signal", "satellite": "signal", "radio": "signal", "broadcast": "signal", "antenna": "signal", "podcast": "signal", "podcasts": "signal", "streaming": "signal",
	"candle": "candle", "candles": "candle", "vigil": "candle",
	"gear": "gears", "gears": "gears", "engine": "gears", "mechanical": "gears", "machinery": "gears", "factory": "gears",
	"bell": "bell", "bells": "bell", "notification": "bell", "subscribe": "bell", "subscribed": "bell",
	"star": "star", "stars": "star", "famous": "star", "celebrity": "star", "favorite": "star", "favourite": "star",
	"thumbs": "thumbs up", "approve": "thumbs up", "approved": "thumbs up", "agree": "thumbs up", "agreed": "thumbs up", "awesome": "thumbs up", "great": "thumbs up",
	"check": "check", "done": "check", "finished": "check", "success": "check", "complete": "check", "completed": "check", "correct": "check", "exactly": "check",
	"link": "share", "share": "share", "shared": "share", "forward": "share",
	"chat": "comment", "message": "comment", "messages": "comment", "texting": "comment", "comment": "comment", "comments": "comment", "tweet": "comment",
	"video": "play", "youtube": "play", "movie": "play", "movies": "play", "film": "play", "netflix": "play",
	// math
	"plus": "plus", "minus": "plus", "math": "plus", "maths": "plus", "arithmetic": "plus", "equation": "plus", "calculate": "plus",
	"infinity": "infinity", "infinite": "infinity", "forever": "infinity", "endless": "infinity", "eternity": "infinity",
	"pi": "pi", "pie": "pi", "circle": "pi",
	"sum": "sigma", "total": "sigma", "summation": "sigma",
	"numbers": "numbers", "count": "numbers", "counting": "numbers", "million": "numbers", "billion": "numbers", "thousand": "numbers", "trillion": "numbers",
	"hundred": "hundred", "percent": "hundred", "perfect": "hundred", "100": "hundred", "100%": "hundred",
	"root": "root", "sqrt": "root", "radical": "root",
	"chart": "pie chart", "graph": "pie chart", "stats": "pie chart", "statistics": "pie chart", "data": "pie chart", "percentage": "pie chart", "survey": "pie chart",
	// flags
	"race": "chequered", "racing": "chequered", "finish": "chequered", "nascar": "chequered", "f1": "chequered", "formula": "chequered",
	"flag": "red flag", "flags": "red flag", "warning": "red flag",
	"pride": "rainbow flag", "lgbt": "rainbow flag", "lgbtq": "rainbow flag", "gay": "rainbow flag",
	"pirate": "pirate", "pirates": "pirate", "treasure": "pirate",
	"america": "usa", "american": "usa", "usa": "usa", "washington": "usa", "texas": "usa", "california": "usa", "florida": "usa",
	"japan": "japan", "japanese": "japan", "tokyo": "japan", "anime": "japan", "sushi": "japan",
	"france": "france", "french": "france", "paris": "france",
	"italy": "italy", "italian": "italy", "rome": "italy", "pizza": "italy", "pasta": "italy",
	"germany": "germany", "german": "germany", "berlin": "germany",
	"ukraine": "ukraine", "ukrainian": "ukraine", "kyiv": "ukraine",
	"britain": "uk", "british": "uk", "england": "uk", "english": "uk", "london": "uk", "uk": "uk", "scotland": "uk",
	"brazil": "brazil", "brazilian": "brazil", "rio": "brazil",
	"canada": "canada", "canadian": "canada", "toronto": "canada", "maple": "canada",
	// nature
	"fire": "fire", "fires": "fire", "flame": "fire", "flames": "fire", "burn": "fire", "burning": "fire", "burned": "fire", "hot": "fire", "campfire": "fire", "wildfire": "fire",
	"rain": "rain", "raining": "rain", "rainy": "rain", "cloud": "rain", "clouds": "rain", "cloudy": "rain", "umbrella": "rain",
	"storm": "storm", "storms": "storm", "thunder": "storm", "thunderstorm": "storm",
	"sun": "sun", "sunny": "sun", "sunshine": "sun", "summer": "sun", "sunrise": "sun", "sunset": "sun",
	"moon": "moon", "night": "moon", "midnight": "moon", "lunar": "moon",
	"ocean": "ocean", "sea": "ocean", "waves": "ocean", "beach": "ocean", "surf": "ocean", "surfing": "ocean", "swim": "ocean", "swimming": "ocean", "boat": "ocean", "sailing": "ocean",
	"rainbow": "rainbow", "rainbows": "rainbow", "colours": "rainbow", "colors": "rainbow",
	"lightning": "bolt", "electric": "bolt", "electricity": "bolt", "energy": "bolt", "power": "bolt", "voltage": "bolt", "shock": "bolt", "zap": "bolt",
	"flower": "flower", "flowers": "flower", "bloom": "flower", "blossom": "flower", "spring": "flower", "garden": "seedling",
	"snow": "snowflake", "snowing": "snowflake", "cold": "snowflake", "freezing": "snowflake", "frozen": "snowflake", "ice": "snowflake", "winter": "snowflake", "frost": "snowflake",
	"tornado": "tornado", "hurricane": "tornado", "twister": "tornado", "cyclone": "tornado", "chaos": "tornado",
	"plant": "seedling", "plants": "seedling", "grow": "seedling", "growing": "seedling", "growth": "seedling", "seed": "seedling", "seeds": "seedling", "farm": "seedling", "farming": "seedling",
	"volcano": "volcano", "lava": "volcano", "eruption": "volcano", "erupt": "volcano", "magma": "volcano",
}

var lookup map[string]int

func init() {
	lookup = map[string]int{}
	// glyph names and aliases (single words only, so "mind" alone doesn't fire)
	for i, d := range glyph.Registry {
		for _, a := range append([]string{d.Name}, d.Aliases...) {
			a = strings.ToLower(a)
			if strings.ContainsAny(a, " ") || len(a) < 3 {
				continue
			}
			if _, ok := lookup[a]; !ok {
				lookup[a] = i
			}
		}
	}
	for w, name := range extra {
		if i := glyph.Find(name); i >= 0 {
			lookup[w] = i
		}
	}
	// words too common to keep firing
	for _, w := range []string{"like", "yes", "ok", "okay", "play", "link", "note", "day", "time", "wave", "run", "check", "done", "great", "ask", "hi", "hey", "float", "seed", "wild", "bet", "band", "film", "bench", "power", "focus", "update"} {
		delete(lookup, w)
	}
	// keep a few deliberately: hello, love, laugh, fire, etc. are fun
}

// Trigger returns the glyph index a spoken word should animate, or -1.
// Punctuation is ignored; "?" fires the raised-hand glyph and bracketed
// sound cues like "[laughter]" or "[music]" fire their glyph.
func Trigger(word string) int {
	w := strings.ToLower(strings.TrimSpace(word))
	if w == "" {
		return -1
	}
	if strings.HasPrefix(w, "[") || strings.HasPrefix(w, "(") {
		inner := strings.Trim(w, "[]()")
		switch {
		case strings.Contains(inner, "laugh"):
			return glyph.Find("joy")
		case strings.Contains(inner, "music"):
			return glyph.Find("music")
		case strings.Contains(inner, "applause"), strings.Contains(inner, "clap"):
			return glyph.Find("clap")
		}
		return -1
	}
	if strings.HasSuffix(w, "?") && len(w) > 3 {
		return glyph.Find("raise")
	}
	w = strings.TrimFunc(w, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '\'' && r != '%' })
	if i, ok := lookup[w]; ok {
		return i
	}
	// simple plural / past tense fallback
	for _, suf := range []string{"s", "es", "ed", "ing"} {
		if strings.HasSuffix(w, suf) {
			if i, ok := lookup[strings.TrimSuffix(w, suf)]; ok {
				return i
			}
		}
	}
	return -1
}

// TriggerCount reports how many words are mapped (for the help screen).
func TriggerCount() int { return len(lookup) }
