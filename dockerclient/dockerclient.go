package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/docker/libswarm/backends"
	"github.com/docker/libswarm/beam"
	"log"
	"os"
	"strings"
)

func main() {
	app := cli.NewApp()
	app.Name = "swarmd"
	app.Usage = "Control a heterogenous distributed system with the Docker API"
	app.Version = "0.0.1"
	app.Flags = []cli.Flag{
		cli.StringFlag{"backend", "debug", "load a backend"},
	}
	app.Action = cmdDaemon
	app.Run(os.Args)
}

func cmdDaemon(c *cli.Context) {
	app := beam.NewServer()
	app.OnLog(beam.Handler(func(msg *beam.Message) error {
		log.Printf("%s\n", strings.Join(msg.Args, " "))
		return nil
	}))
	app.OnError(beam.Handler(func(msg *beam.Message) error {
		Fatalf("Fatal: %v", strings.Join(msg.Args[:1], ""))
		return nil
	}))

	backend := beam.Object{backends.Forward()}

	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		dockerHost = "unix:///var/run/docker.sock"
	}

	log.Printf("---> Spawning\n")
	instance, err := backend.Spawn(dockerHost)
	if err != nil {
		Fatalf("spawn: %v\n", err)
	}

	log.Printf("---> Attaching\n")
	instanceIn, instanceOut, err := instance.Attach("")
	if err != nil {
		Fatalf("attach: %v", err)
	}
	defer instanceOut.Close()
	go beam.Copy(app, instanceIn)

	log.Printf("---> Starting\n")
	if err := instance.Start(); err != nil {
		Fatalf("start: %v", err)
	}

	err = doCmd(instance, c.Args())
	if err != nil {
		Fatalf("%v", err)
	}
}

func doCmd(instance *beam.Object, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command supplied")
	}
	log.Printf("---> %s\n", args[0])
	if args[0] == "ps" {
		if len(args) != 1 {
			return fmt.Errorf("usage: ps")
		}
		names, err := instance.Ls()
		if err != nil {
			return err
		}
		fmt.Println(strings.Join(names, "\n"))
		return nil
	}
	return fmt.Errorf("unrecognised command: %s", args[0])
}

func Fatalf(msg string, args ...interface{}) {
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	fmt.Fprintf(os.Stderr, msg, args...)
	os.Exit(1)
}