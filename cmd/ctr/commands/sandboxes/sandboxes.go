package sandboxes

import (
	"fmt"
	"github.com/containerd/containerd"
	"github.com/containerd/containerd/cmd/ctr/commands"
	"github.com/containerd/containerd/defaults"
	"github.com/containerd/containerd/errdefs"
	"github.com/containerd/containerd/oci"
	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"os"
	"text/tabwriter"
)

var (
	sandboxerFlages = []cli.Flag{
		cli.StringFlag{
			Name:  "sandboxer",
			Usage: "sandboxer name, default value is runc",
			Value: defaults.DefaultSandboxer,
		},
	}
	createFlags = []cli.Flag{
		cli.StringFlag{
			Name:  "config,c",
			Usage: "path to the runtime-specific spec config file",
		},
		cli.StringFlag{
			Name:  "cwd",
			Usage: "specify the working directory of the process",
		},
		cli.StringSliceFlag{
			Name:  "env",
			Usage: "specify additional container environment variables (e.g. FOO=bar)",
		},
		cli.StringFlag{
			Name:  "env-file",
			Usage: "specify additional container environment variables in a file(e.g. FOO=bar, one per line)",
		},
		cli.StringSliceFlag{
			Name:  "label",
			Usage: "specify additional labels (e.g. foo=bar)",
		},
		cli.BoolFlag{
			Name:  "net-host",
			Usage: "enable host networking for the container",
		},
		cli.BoolFlag{
			Name:  "privileged",
			Usage: "run privileged container",
		},
		cli.BoolFlag{
			Name:  "read-only",
			Usage: "set the containers filesystem as readonly",
		},
		cli.StringFlag{
			Name:  "sandboxer",
			Usage: "sandboxer name",
			Value: defaults.DefaultSandboxer,
		},
		cli.StringFlag{
			Name:  "runtime-config-path",
			Usage: "optional runtime config path",
		},
		cli.StringSliceFlag{
			Name:  "with-ns",
			Usage: "specify existing Linux namespaces to join at container runtime (format '<nstype>:<path>')",
		},
		cli.StringFlag{
			Name:  "pid-file",
			Usage: "file path to write the task's pid",
		},
		cli.IntFlag{
			Name:  "gpus",
			Usage: "add gpus to the container",
		},
		cli.BoolFlag{
			Name:  "allow-new-privs",
			Usage: "turn off OCI spec's NoNewPrivileges feature flag",
		},
		cli.Uint64Flag{
			Name:  "memory-limit",
			Usage: "memory limit (in bytes) for the container",
		},
		cli.StringSliceFlag{
			Name:  "device",
			Usage: "file path to a device to add to the container; or a path to a directory tree of devices to add to the container",
		},
		cli.BoolFlag{
			Name:  "seccomp",
			Usage: "enable the default seccomp profile",
		},
		cli.StringFlag{
			Name:  "seccomp-profile",
			Usage: "file path to custom seccomp profile. seccomp must be set to true, before using seccomp-profile",
		},
		cli.StringFlag{
			Name:  "apparmor-default-profile",
			Usage: "enable AppArmor with the default profile with the specified name, e.g. \"cri-containerd.apparmor.d\"",
		},
		cli.StringFlag{
			Name:  "apparmor-profile",
			Usage: "enable AppArmor with an existing custom profile",
		},
	}
)

// Command is the cli command for managing containers
var Command = cli.Command{
	Name:    "sandboxes",
	Usage:   "manage sandboxes",
	Aliases: []string{"s", "sandbox"},
	Flags:   sandboxerFlages,
	Subcommands: []cli.Command{
		createCommand,
		deleteCommand,
		//infoCommand,
		listCommand,
	},
}

var createCommand = cli.Command{
	Name:      "create",
	Usage:     "create sandbox",
	ArgsUsage: "[flags] SANDBOX",
	Flags:     createFlags,
	Action: func(context *cli.Context) error {
		sandboxer := context.GlobalString("sandboxer")
		id := context.Args().First()
		if context.NArg() != 1 {
			return errors.Wrap(errdefs.ErrInvalidArgument, "only sandbox id should be provided")
		}
		if id == "" {
			return errors.Wrap(errdefs.ErrInvalidArgument, "sandbox id must be provided")
		}
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		var specOpts []containerd.SimpleSpecOpts
		if context.IsSet("config") {
			specOpts = append(specOpts, containerd.Simple(oci.WithSpecFromFile(context.String("config"))))
		} else {
			specOpts = append(specOpts, containerd.Simple(oci.WithDefaultSpec()))
			if context.Bool("net-host") {
				specOpts = append(specOpts, containerd.Simple(oci.WithHostNamespace(specs.NetworkNamespace)), containerd.Simple(oci.WithHostHostsFile), containerd.Simple(oci.WithHostResolvconf))
			}
			if cpus := context.Float64("cpus"); cpus > 0.0 {
				var (
					period = uint64(100000)
					quota  = int64(cpus * 100000.0)
				)
				specOpts = append(specOpts, containerd.Simple(oci.WithCPUCFS(quota, period)))
			}

			quota := context.Int64("cpu-quota")
			period := context.Uint64("cpu-period")
			if quota != -1 || period != 0 {
				if cpus := context.Float64("cpus"); cpus > 0.0 {
					return errors.New("cpus and quota/period should be used separately")
				}
				specOpts = append(specOpts, containerd.Simple(oci.WithCPUCFS(quota, period)))
			}
			limit := context.Uint64("memory-limit")
			if limit != 0 {
				specOpts = append(specOpts, containerd.Simple(oci.WithMemoryLimit(limit)))
			}
		}
		var opts []containerd.NewSandboxOpt
		var s specs.Spec
		opts = append(opts, containerd.WithSandboxSpec(&s, specOpts...))
		_, err = client.NewSandbox(ctx, sandboxer, id, opts...)
		return err
	},
}

var deleteCommand = cli.Command{
	Name:      "delete",
	Usage:     "delete sandbox",
	ArgsUsage: "[flags] SANDBOX",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "force",
			Usage: "force delete sandbox even there are containers",
		},
	},
	Action: func(context *cli.Context) error {
		sandboxer := context.GlobalString("sandboxer")
		id := context.Args().First()
		if context.NArg() != 1 {
			return errors.Wrap(errdefs.ErrInvalidArgument, "only sandbox id should be provided")
		}
		if id == "" {
			return errors.Wrap(errdefs.ErrInvalidArgument, "sandbox id must be provided")
		}
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		sandboxInstance, err := client.LoadSandbox(ctx, sandboxer, id)
		if err != nil {
			return err
		}
		sandbox, err := sandboxInstance.Metadata(ctx)
		if err != nil {
			return err
		}

		if len(sandbox.Containers) > 0 && !context.Bool("force") {
			return fmt.Errorf("there are containers in sandbox, remove them first")
		}
		if err := sandboxInstance.Delete(ctx); err != nil {
			if !errdefs.IsNotFound(err) {
				return err
			}
		}
		return nil
	},
}

var listCommand = cli.Command{
	Name:      "list",
	Aliases:   []string{"ls"},
	Usage:     "list sandboxes",
	ArgsUsage: "[flags] [<filter>, ...]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "print only the sandbox id",
		},
	},
	Action: func(context *cli.Context) error {
		var (
			filters       = context.Args()
			quiet         = context.Bool("quiet")
			sandboxerName = context.GlobalString("sandboxer")
		)
		client, ctx, cancel, err := commands.NewClient(context)
		if err != nil {
			return err
		}
		defer cancel()
		sandboxer := client.SandboxService(sandboxerName)
		sandboxes, err := sandboxer.List(ctx, filters...)
		if err != nil {
			return err
		}
		if quiet {
			for _, s := range sandboxes {
				fmt.Printf("%s\n", s.ID)
			}
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 4, 8, 4, ' ', 0)
		fmt.Fprintln(w, "SANDBOX\tSTATUS\tCONTAINERS\tADDRESS\t")
		for _, s := range sandboxes {
			statusStr := "unknown"
			status, err := sandboxer.Status(ctx, s.ID)
			if err != nil {
				statusStr = "error"
			} else {
				statusStr = string(status.State)
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%d\t%s\t\n",
				s.ID,
				statusStr,
				len(s.Containers),
				s.TaskAddress,
			); err != nil {
				return err
			}
		}
		return w.Flush()
	},
}
