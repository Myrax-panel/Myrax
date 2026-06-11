package version

var Version = "0.1.1"

const GitHubReleasesAPI = "https://api.github.com/repos/Myrax-panel/Myrax/releases"

const GitHubReleasesPage = "https://github.com/Myrax-panel/Myrax/releases/latest"

func GitHubLatestDownloadURL(arch string) string {
	return "https://github.com/Myrax-panel/Myrax/releases/latest/download/myrax-linux-" + arch
}
