module github.com/ISE-SMILE/SET

go 1.16

require (
	github.com/aws/aws-sdk-go v1.37.22
	github.com/faas-facts/bench v0.0.1
	github.com/faas-facts/fact v0.1.5
	github.com/faas-facts/fact-go-client v0.1.6
	github.com/google/martian v2.1.0+incompatible
	github.com/sirupsen/logrus v1.8.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/x-cray/logrus-prefixed-formatter v0.5.2
	gopkg.in/yaml.v3 v3.0.1
)

replace github.com/faas-facts/bench v0.0.1 => ../fact-bench
