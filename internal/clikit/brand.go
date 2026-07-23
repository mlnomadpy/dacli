package clikit

// Tagline is dacli's single positioning line. It appears identically in the
// README hero, docs/index.md hero, and here in the CLI banner — one brand,
// not three copies that drift.
const Tagline = "Your autonomous engineering team — set the direction; it plans, builds, reviews, and ships."

// bannerArt is a restrained ASCII rendition of the hexagon-cluster mark in
// docs/assets/logo.svg: a coordinated grid of units, not a chaotic swarm.
// Stdlib string art only — no figlet, no new dependency.
const bannerArt = `  ◆───◆───◆
  │   │   │
  ◆───◆───◆   dacli
  │   │   │
  ◆───◆───◆`

// Banner renders the mark plus the tagline, for the two human entry points
// that carry the brand: bare "dacli" and "dacli version".
func Banner() string {
	return bannerArt + "\n\n" + Tagline + "\n\n"
}
