package project

var (
	description = "The aws-rolling-node-operator refreshes instances on ASG's."
	gitSHA      = "n/a"
	name        = "aws-rolling-node-operator"
	source      = "https://github.com/giantswarm/aws-rolling-node-operator"
	version     = "0.3.2"
)

func Description() string {
	return description
}

func GitSHA() string {
	return gitSHA
}

func Name() string {
	return name
}

func Source() string {
	return source
}

func Version() string {
	return version
}
