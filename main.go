/*
 * Copyright (C) 2021.   Sebastian Werner, TU Berlin, Germany
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */
package main

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ISE-SMILE/SET/set"
	"gopkg.in/yaml.v3"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/faas-facts/bench/bencher"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

const LICENCE_TEXT = "Copyright (C) 2021 Sebastian Werner\nThis program comes with ABSOLUTELY NO WARRANTY; GNU GPLv3"

var (
	Build string
)

var logger = logrus.New()
var log *logrus.Entry

func init() {
	if Build == "" {
		Build = "Debug"
	}
	logger.Formatter = new(prefixed.TextFormatter)
	logger.SetLevel(logrus.DebugLevel)
	log = logger.WithFields(logrus.Fields{
		"prefix": "set",
		"build":  Build,
	})
}

func setup() {
	fmt.Println(LICENCE_TEXT)

	viper.SetConfigName("set")
	viper.AddConfigPath(".")

	//setup defaults
	viper.SetDefault("verbose", true)
	viper.SetDefault("unattended", false)
	viper.SetDefault("workload", "examples/workload.yml")

	//setup cmd interface
	flag.Bool("verbose", false, "for verbose logging")
	flag.String("workload", "workloads/b0.yml", "the workload descriptor file")
	flag.Bool("y", false, "run without waiting for user confirmation")

	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()

	viper.RegisterAlias("y", "unattended")

	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		log.Errorf("error parsing flags %+v", err)
	}

	if viper.GetBool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
	}
}

func main() {
	setup()
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	runtime.GOMAXPROCS(runtime.NumCPU())

	if viper.GetBool("verbose") {
		logger.SetLevel(logrus.DebugLevel)
		bencher.SetDefaultLogger(log)
	}

	w := set.PerformanceWorkload{}
	worklaodFile := viper.GetString("workload")
	data, err := os.ReadFile(worklaodFile)
	if err != nil {
		panic(err)
	}
	if strings.HasSuffix(worklaodFile, "yml") || strings.HasSuffix(worklaodFile, "yaml") {
		err = yaml.Unmarshal(data, &w)
		if err != nil {
			panic(err)
		}
	} else if strings.HasSuffix(worklaodFile, "json") {
		err = json.Unmarshal(data, &w)
		if err != nil {
			panic(err)
		}
	} else {
		panic(fmt.Sprintf("cant read worklaod file type - %s", worklaodFile))
	}

	platform := set.MakefileDeployment{}

	w.Platform = platform

	bench := w.Prepare()

	if !bencher.AskForConfirmation("deploying the workload", os.Stdin) {
		os.Exit(0)
	}

	target, err := platform.Deploy(w.Deployment)
	if err != nil {
		panic(err)
	}

	if w.Target == "" {
		w.Target = target
	}
	if w.Type == "io" {
		if !bencher.AskForConfirmation("generating IO Objects?", os.Stdin) {
			os.Exit(0)
		}
		w.GenerateIObjects()
	}

	if !bencher.AskForConfirmation("run SET benchmark?", os.Stdin) {
		os.Exit(0)
	}
	bench.Run()

}
