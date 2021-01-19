package powershell

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"

	"github.com/thxcode/terraform-provider-windbag/windbag/utils"
)

type StdStream func(output interface{})

type ExecutorName string
type IOFormat string
type WindowStyle string
type ExecutionPolicy string

const (
	ExecutorPowerShell ExecutorName = "powershell.exe"
	ExecutorPWSH       ExecutorName = "pwsh.exe"

	// refer: https://docs.microsoft.com/en-us/powershell/module/Microsoft.PowerShell.Core/About/about_PowerShell_exe?view=powershell-5.1#-inputformat-text--xml
	IOFormatText IOFormat = "Text"
	IOFormatXML  IOFormat = "XML"

	// refer: https://docs.microsoft.com/en-us/powershell/module/Microsoft.PowerShell.Core/About/about_execution_policies?view=powershell-5.1#powershell-execution-policies
	ExecutionPolicyDefault      ExecutionPolicy = "Default"
	ExecutionPolicyAllSigned    ExecutionPolicy = "AllSigned"
	ExecutionPolicyBypass       ExecutionPolicy = "Bypass"
	ExecutionPolicyRemoteSigned ExecutionPolicy = "RemoteSigned"
	ExecutionPolicyRestricted   ExecutionPolicy = "Restricted"
	ExecutionPolicyUndefined    ExecutionPolicy = "Undefined"
	ExecutionPolicyUnrestricted ExecutionPolicy = "Unrestricted"
)

type CreateOptions struct {
	Executor          ExecutorName    // executor of PowerShell.
	Sta               bool            // starts PowerShell using a single-threaded apartment.
	NoProfile         bool            // does not load the PowerShell profile.
	InputFormat       IOFormat        // describes the format of data sent to PowerShell.
	OutputFormat      IOFormat        // determines how output from PowerShell is formatted.
	ConfigurationName string          // specifies a configuration endpoint in which PowerShell is run.
	ExecutionPolicy   ExecutionPolicy // sets the default execution policy for the current session and saves it in the `$env:PSExecutionPolicyPreference` environment variable.
}

func Create(session *ssh.Session, opts *CreateOptions) *PowerShell {
	if opts == nil {
		opts = &CreateOptions{}
	}

	var args = make([]string, 0, 32)
	if opts.Executor == "" {
		opts.Executor = ExecutorPowerShell
	}
	args = append(args, string(opts.Executor))
	if opts.Sta {
		args = append(args, "-Sta")
	}
	if opts.NoProfile {
		args = append(args, "-NoProfile")
	}
	if opts.InputFormat != "" {
		args = append(args, "-InputFormat", string(opts.InputFormat))
	}
	if opts.OutputFormat != "" {
		args = append(args, "-OutputFormat", string(opts.OutputFormat))
	}
	if opts.ConfigurationName != "" {
		args = append(args, "-ConfigurationName", opts.ConfigurationName)
	}
	if opts.ExecutionPolicy != "" {
		args = append(args, "-ExecutionPolicy", string(opts.ExecutionPolicy))
	}

	return &PowerShell{
		args:    args,
		session: session,
	}
}

type PowerShell struct {
	executed bool
	args     []string
	session  *ssh.Session
}

// ExecuteScript executes the `scriptPath` script with `scriptArgs`, this method will be blocked until finish or error occur,
// returns nil when exit code is 0
func (ps *PowerShell) ExecuteScript(ctx context.Context, id string, stdout, stderr StdStream, scriptPath string, scriptArgs ...string) error {
	if len(scriptPath) == 0 {
		return errors.New("can't exec blank script")
	}
	log.Printf("[INFO] [PowerShell -(%s)- Stdin]: %s, %v", id, scriptPath, scriptArgs)

	if ps.executed {
		return errors.New("cannot re-execute the powershell")
	}
	ps.executed = true

	// prepare
	var args = append(ps.args, "-NoLogo", "-NonInteractive", "-WindowStyle", "Hidden", "-File", scriptPath)
	args = append(args, scriptArgs...)

	var session = ps.session
	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stdout stream")
	}
	sessionStderr, err := session.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stderr stream")
	}

	var eg errgroup.Group
	eg.Go(func() error {
		defer utils.HandleCrash()
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = sessionStdout.Read(buf)
			if readSize > 0 {
				var ret = string(buf[:readSize])
				if stdout != nil {
					stdout(ret)
				}
				log.Printf("[DEBUG] [PowerShell -(%s)- Stdout]: %s\n", id, ret)
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	eg.Go(func() error {
		defer utils.HandleCrash()
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = sessionStderr.Read(buf)
			if readSize > 0 {
				ret := string(buf[:readSize])
				if stderr != nil {
					stderr(ret)
				}
				log.Printf("[WARN] [PowerShell -(%s)- Stderr]: %s\n", id, ret)
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	eg.Go(func() error {
		defer utils.HandleCrash()
		if err := session.Run(strings.Join(args, " ")); err != nil {
			return errors.Wrapf(err, "could not execute script %s", scriptPath)
		}
		return nil
	})
	return eg.Wait()
}

// ExecuteCommand executes the `command`, this method will be blocked until finish or error occur,
// returns nil when exit code is 0.
func (ps *PowerShell) ExecuteCommand(ctx context.Context, id string, stdout, stderr StdStream, command string) error {
	if len(command) == 0 {
		return errors.New("can't exec blank command")
	}
	log.Printf("[INFO] [PowerShell -(%s)- Stdin]: %s", id, command)

	if ps.executed {
		return errors.New("cannot re-execute the powershell")
	}
	ps.executed = true

	// prepare
	var args = append(ps.args, "-NoLogo", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", "$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue';", command)

	var session = ps.session
	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stdout stream")
	}
	sessionStderr, err := session.StderrPipe()
	if err != nil {
		return errors.Wrap(err, "could not take over the PowerShell's stderr stream")
	}

	var eg errgroup.Group
	eg.Go(func() error {
		defer utils.HandleCrash()
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = sessionStdout.Read(buf)
			if readSize > 0 {
				var ret = string(buf[:readSize])
				if stdout != nil {
					stdout(ret)
				}
				log.Printf("[DEBUG] [PowerShell -(%s)- Stdout]: %s\n", id, ret)
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	eg.Go(func() error {
		defer utils.HandleCrash()
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = sessionStderr.Read(buf)
			if readSize > 0 {
				var ret = string(buf[:readSize])
				if stderr != nil {
					stderr(ret)
				}
				log.Printf("[WARN] [PowerShell -(%s)- Stderr]: %s\n", id, ret)
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	eg.Go(func() error {
		defer utils.HandleCrash()
		if err := session.Run(strings.Join(args, " ")); err != nil {
			return errors.Wrapf(err, "could not execute command %s", command)
		}
		return nil
	})
	return eg.Wait()
}

// Commands holds the input of PowerShell.
func (ps *PowerShell) Commands() (*Commands, error) {
	if ps.executed {
		return nil, errors.New("cannot re-execute the powershell")
	}
	ps.executed = true

	// prepare
	var args = append(ps.args, "-NoLogo", "-NonInteractive", "-NoExit", "-WindowStyle", "Hidden", "-Command", "-")

	var session = ps.session
	sessionStdin, err := session.StdinPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stdin stream")
	}
	sessionStdout, err := session.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stdout stream")
	}
	sessionStderr, err := session.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "could not take over the PowerShell's stderr stream")
	}
	err = session.Start(strings.Join(args, " "))
	if err != nil {
		return nil, errors.Wrap(err, "could not spawn PowerShell process")
	}

	return &Commands{
		session:       session,
		sessionStdin:  sessionStdin,
		sessionStdout: sessionStdout,
		sessionStderr: sessionStderr,
	}, nil
}

type Commands struct {
	session       *ssh.Session
	sessionStdin  io.WriteCloser
	sessionStdout io.Reader
	sessionStderr io.Reader
}

// Execute allows to input a `command` one by one, returns execution result, stdout info, stderr info and error.
func (psc *Commands) Execute(ctx context.Context, id string, command string) (string, string, error) {
	if len(command) == 0 {
		return "", "", errors.New("could not execute blank cmd")
	}
	log.Printf("[INFO] [PowerShell -(%s)- Stdin]: %s", id, command)
	command = strings.Replace(command, "\n", " ", -1) // narrow the command into one line

	var commandSignal = newCommandSignal()
	var commandWrapper = fmt.Sprintf("$ErrorActionPreference='Stop'; $ProgressPreference='SilentlyContinue'; Try {%s} Catch {[System.Console]::Error.Write($_.Exception.Message)}; [System.Console]::Out.Write(\"%s\"); [System.Console]::Error.Write(\"%s\");\r\n", command, commandSignal, commandSignal)
	_, err := psc.sessionStdin.Write([]byte(commandWrapper))
	if err != nil {
		return "", "", errors.Errorf("could not input %q command into PowerShell stdin stream", commandWrapper)
	}

	var (
		commandStdout = &strings.Builder{}
		commandStderr = &strings.Builder{}
		eg            errgroup.Group
	)
	eg.Go(func() error {
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = psc.sessionStdout.Read(buf)
			if readSize > 0 {
				var ret = strings.TrimSuffix(string(buf[:readSize]), commandSignal)
				if ret != "" {
					commandStdout.WriteString(ret)
					log.Printf("[DEBUG] [PowerShell -(%s)- Stdout]: %s\n", id, ret)
				}
				if len(ret) != readSize {
					break
				}
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	eg.Go(func() error {
		var buf = make([]byte, 1<<10)
		for {
			var readSize, err = psc.sessionStderr.Read(buf)
			if readSize > 0 {
				var ret = strings.TrimSuffix(string(buf[:readSize]), commandSignal)
				if ret != "" {
					commandStderr.WriteString(ret)
					log.Printf("[WARN] [PowerShell -(%s)- Stderr]: %s\n", id, ret)
				}
				if len(ret) != readSize {
					break
				}
			}
			if err != nil {
				if io.EOF != err && io.ErrClosedPipe != err {
					return err
				}
				break
			}
		}
		return nil
	})
	if err := eg.Wait(); err != nil {
		return "", "", errors.Wrapf(err, "could not execute command %s", command)
	}
	return commandStdout.String(), commandStderr.String(), nil
}

func (psc *Commands) Close() error {
	_, err := psc.sessionStdin.Write([]byte(`exit\r\n`))
	if err != nil {
		return err
	}

	err = psc.sessionStdin.Close()
	if err != nil {
		return err
	}

	err = psc.session.Wait()
	if err != nil {
		if _, ok := err.(*ssh.ExitError); ok {
			return nil
		}
	}
	return nil
}

func newCommandSignal() string {
	var randArr = make([]byte, 8)
	_, _ = rand.Read(randArr)
	return fmt.Sprintf("#%s#", hex.EncodeToString(randArr))
}
