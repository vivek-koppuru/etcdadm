package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"sigs.k8s.io/etcdadm/apis"
	"sigs.k8s.io/etcdadm/initsystem"
	log "sigs.k8s.io/etcdadm/pkg/logrus"
)

type phaseInput struct {
	initSystem    initsystem.InitSystem
	etcdAdmConfig *apis.EtcdAdmConfig
}

type runFunc func(*phaseInput) error

type phase interface {
	name() string
	run(*phaseInput) error
	registerInCommand(cmd *cobra.Command, runner *runner)
}

type singlePhase struct {
	phaseName     string
	runFunc       runFunc
	prerequisites []phase
}

func (p *singlePhase) name() string {
	return p.phaseName
}

func (p *singlePhase) run(phaseInput *phaseInput) error {
	return p.runFunc(phaseInput)
}

func (p *singlePhase) registerInCommand(cmd *cobra.Command, runner *runner) {
	phaseCmd := &cobra.Command{
		Use:   p.phaseName,
		Short: fmt.Sprintf("Run %s phase", p.phaseName),
		Run: func(cmd *cobra.Command, args []string) {
			phases := make([]phase, 0, len(p.prerequisites)+1)
			phases = append(phases, p.prerequisites...)
			phases = append(phases, p)
			if err := runner.runPhases(cmd, args, phases...); err != nil {
				log.Fatal(err)
			}
		},
	}
	cmd.AddCommand(phaseCmd)
}

type initFunc func(cmd *cobra.Command, args []string) (*phaseInput, error)

type runner struct {
	phases []phase
	init   initFunc
}

func newRunner(init initFunc) *runner {
	return &runner{
		phases: make([]phase, 0),
		init:   init,
	}
}

func (r *runner) registerPhases(phases ...phase) {
	r.phases = append(r.phases, phases...)
}

func (r *runner) run(cmd *cobra.Command, args []string) error {
	return r.runPhases(cmd, args, r.phases...)
}

func (r *runner) runPhases(cmd *cobra.Command, args []string, phases ...phase) error {
	phaseInput, err := r.init(cmd, args)
	if err != nil {
		return err
	}

	for _, phase := range phases {
		if err := phase.run(phaseInput); err != nil {
			return fmt.Errorf("[%s] %s", phase.name(), err)
		}
	}

	return nil
}

func (r *runner) registerPhasesAsSubcommands(cmd *cobra.Command) {
	phaseCmd := &cobra.Command{
		Use:   "phase",
		Short: fmt.Sprintf("Use this command to invoke single phase of the %s command", cmd.Name()),
	}

	for _, phase := range r.phases {
		phase.registerInCommand(phaseCmd, r)
	}

	cmd.AddCommand(phaseCmd)
}
