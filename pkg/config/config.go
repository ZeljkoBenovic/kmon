package config

import (
	"log/slog"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Runner interface {
	PodCmdHandler() error
	PVCCmdHandler() error
}

type Config struct {
	rootCmd *cobra.Command
	podCmd  *cobra.Command
	pvcCmd  *cobra.Command

	log        *slog.Logger
	configPath string

	Context   string `mapstructure:"context"`
	Namespace string `mapstructure:"namespace"`
	Pod       Pod    `mapstructure:"pod"`
	PVC       PVC    `mapstructure:"pvc"`
}

type PodOperationMode string
type PVCOperationMode string

var (
	RunFromPVC      PodOperationMode = "run-from-pvc"
	RunFromSnapshot PodOperationMode = "run-from-snapshot"
)

var (
	SnapshotFromPVC PVCOperationMode = "snapshot-from-pvc"
)

func (p *PodOperationMode) stringPtr() *string {
	return (*string)(p)
}

func (s *PVCOperationMode) stringPtr() *string {
	return (*string)(s)
}

type Pod struct {
	Mode         PodOperationMode `mapstructure:"mode"`
	Name         string           `mapstructure:"name"`
	MountPath    string           `mapstructure:"mount_path"`
	VolumeName   string           `mapstructure:"volume_name"`
	PVCName      string           `mapstructure:"pvc_name"`
	SnapshotName string           `mapstructure:"snapshot_name"`
}

type PVC struct {
	Mode              PVCOperationMode `mapstructure:"mode"`
	Name              string           `mapstructure:"name"`
	SnapshotClassName string           `mapstructure:"snapshot_class_name"`
	SourcePVCName     string           `mapstructure:"source_pvc_name"`
	SnapshotName      string           `mapstructure:"snapshot_name"`
}

func NewConfig(log *slog.Logger) (*Config, error) {
	var c Config

	c.log = log
	c.rootCmd = &cobra.Command{
		Use: "kmon",
		Long: `======[KMON]======
Kmon is a CLI tool to help automate some of the common Kubernetes tasks. 
Some examples:
* deploy a pod with a specified PVC and list its content
* deploy a pod with a PVC restored from a specific snapshot and check its content
* create a snapshot of a specified PVC
and the list goes on...
`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if c.rootCmd.PersistentFlags().Changed("config") {
				if err := viper.BindPFlag("config", c.rootCmd.PersistentFlags().Lookup("config")); err != nil {
					return err
				}

				viper.SetConfigFile(viper.GetString("config"))
				if err := viper.ReadInConfig(); err != nil {
					return err
				}

				c.log.Info("using config file", "file", viper.ConfigFileUsed())

				return viper.Unmarshal(&c)
			}

			return nil
		},
	}

	c.podCmd = &cobra.Command{
		Use:     "pod",
		Long:    "Kubernetes operations on pods",
		Example: "kmon pod --mode run-from-pvc --pvc-name test-pvc",
	}

	c.pvcCmd = &cobra.Command{
		Use:     "pvc",
		Long:    "Kubernetes operations on PVCs",
		Example: "kmon pvc --mode snapshot-from-pvc --source-pvc-name test-pvc",
	}

	c.rootCmd.AddCommand(c.podCmd)
	c.rootCmd.AddCommand(c.pvcCmd)

	return &c, nil
}

func (c *Config) Execute(handlers Runner) error {
	c.rootCmd.PersistentFlags().StringVarP(&c.configPath, "config", "c", "", "path to config file")
	c.rootCmd.PersistentFlags().StringVarP(&c.Namespace, "namespace", "n", "default", "namespace to run in")
	c.rootCmd.PersistentFlags().StringVar(&c.Context, "context", "", "context to run in")

	pf := c.podCmd.Flags()
	pf.StringVar(c.Pod.Mode.stringPtr(), "mode", "", "pod operation mode")
	pf.StringVar(&c.Pod.Name, "name", "kmon-pod", "pod name")
	pf.StringVar(&c.Pod.VolumeName, "volume-name", "kmon-volume", "volume name")
	pf.StringVar(&c.Pod.MountPath, "mount-path", "kmon-mnt", "mount path")
	pf.StringVar(&c.Pod.PVCName, "pvc-name", "kmon-pvc", "pvc name")
	pf.StringVar(&c.Pod.SnapshotName, "snapshot-name", "kmon-snapshot", "snapshot name")
	_ = viper.BindPFlag("pod.mode", pf.Lookup("mode"))
	_ = viper.BindPFlag("pod.name", pf.Lookup("name"))
	_ = viper.BindPFlag("pod.volume-name", pf.Lookup("volume-name"))
	_ = viper.BindPFlag("pod.mount-path", pf.Lookup("mount-path"))
	_ = viper.BindPFlag("pod.pvc-name", pf.Lookup("pvc-name"))
	_ = viper.BindPFlag("pod.snapshot-name", pf.Lookup("snapshot-name"))

	pvf := c.pvcCmd.Flags()
	pvf.StringVar(c.PVC.Mode.stringPtr(), "mode", "", "pod operation mode")
	pvf.StringVar(&c.PVC.Name, "name", "kmon-pvc", "pvc name")
	pvf.StringVar(&c.PVC.SnapshotClassName, "snapshot-class-name", "", "snapshot class name")
	pvf.StringVar(&c.PVC.SourcePVCName, "source-pvc-name", "", "source pvc name")
	pvf.StringVar(&c.PVC.SnapshotName, "snapshot-name", "kmon-snap", "snapshot name")
	_ = viper.BindPFlag("pvc.mode", pf.Lookup("mode"))
	_ = viper.BindPFlag("pvc.name", pvf.Lookup("name"))
	_ = viper.BindPFlag("pvc.snapshot-class-name", pvf.Lookup("source-class-name"))
	_ = viper.BindPFlag("pvc.source-pvc-name", pvf.Lookup("source-pvc-name"))
	_ = viper.BindPFlag("pvc.snapshot-name", pvf.Lookup("snapshot-name"))

	c.rootCmd.RunE = func(_ *cobra.Command, _ []string) error { return c.rootCmd.Help() }
	c.podCmd.RunE = func(_ *cobra.Command, _ []string) error { return handlers.PodCmdHandler() }
	c.pvcCmd.RunE = func(_ *cobra.Command, _ []string) error { return handlers.PVCCmdHandler() }

	return c.rootCmd.Execute()
}
