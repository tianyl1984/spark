package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Fixed file names inside each project's config folder under $HOME/.spark.
const (
	executeFile = "execute.sh"
	workdirFile = "workdir"
	successFile = "success.sh"
	failFile    = "fail.sh"
)

// projectFiles are all fixed file names created when scaffolding a project.
var projectFiles = []string{executeFile, workdirFile, successFile, failFile}

// Scaffold creates the project folder under sparkDir and the fixed (empty)
// config files. Existing files are left untouched. It returns the paths that
// were newly created.
func Scaffold(sparkDir, project string) ([]string, error) {
	dir := filepath.Join(sparkDir, project)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create %s: %w", dir, err)
	}

	var created []string
	for _, name := range projectFiles {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			continue // already exists, don't clobber
		}
		mode := os.FileMode(0o644)
		if strings.HasSuffix(name, ".sh") {
			mode = 0o755
		}
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, mode)
		if err != nil {
			return created, fmt.Errorf("create %s: %w", path, err)
		}
		_ = f.Close()
		created = append(created, path)
	}
	return created, nil
}

// Runner executes per-project scripts located under the spark config directory.
type Runner struct {
	// SparkDir is the root config directory ($HOME/.spark).
	SparkDir string
}

// New creates a Runner rooted at sparkDir.
func New(sparkDir string) *Runner {
	return &Runner{SparkDir: sparkDir}
}

// HasProject reports whether a config folder with an execute.sh exists for project.
func (r *Runner) HasProject(project string) bool {
	info, err := os.Stat(filepath.Join(r.SparkDir, project, executeFile))
	return err == nil && !info.IsDir()
}

// Run executes the project's execute.sh, then runs success.sh or fail.sh
// depending on the outcome. The combined output of execute.sh is passed to the
// follow-up script as a single argument.
//
// Run is intended to be called in its own goroutine; it logs progress and
// returns the error from execute.sh (nil on success).
func (r *Runner) Run(ctx context.Context, project string) error {
	projectDir := filepath.Join(r.SparkDir, project)

	executePath := filepath.Join(projectDir, executeFile)
	if info, err := os.Stat(executePath); err != nil || info.IsDir() {
		return fmt.Errorf("execute.sh not found for project %q", project)
	}

	workdir := r.resolveWorkdir(projectDir)

	log.Printf("[%s] running %s (workdir=%s)", project, executeFile, workdir)
	output, runErr := runScript(ctx, executePath, workdir)
	log.Printf("[%s] execute.sh output:\n%s", project, strings.TrimRight(output, "\n"))

	if runErr != nil {
		log.Printf("[%s] execute.sh failed: %v", project, runErr)
		r.runHook(ctx, project, projectDir, failFile, workdir, output)
		return runErr
	}

	log.Printf("[%s] execute.sh succeeded", project)
	r.runHook(ctx, project, projectDir, successFile, workdir, output)
	return nil
}

// resolveWorkdir reads the workdir file; falls back to the project dir when the
// file is missing or empty.
func (r *Runner) resolveWorkdir(projectDir string) string {
	data, err := os.ReadFile(filepath.Join(projectDir, workdirFile))
	if err != nil {
		return projectDir
	}
	wd := strings.TrimSpace(string(data))
	if wd == "" {
		return projectDir
	}
	return wd
}

// runHook runs success.sh / fail.sh if it exists and is non-empty, passing the
// execute.sh output as a single argument.
func (r *Runner) runHook(ctx context.Context, project, projectDir, name, workdir, output string) {
	path := filepath.Join(projectDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || info.Size() == 0 {
		log.Printf("[%s] skipping %s (missing or empty)", project, name)
		return
	}

	log.Printf("[%s] running %s", project, name)
	hookOut, hookErr := runScript(ctx, path, workdir, output)
	if hookErr != nil {
		log.Printf("[%s] %s failed: %v\n%s", project, name, hookErr, strings.TrimRight(hookOut, "\n"))
		return
	}
	log.Printf("[%s] %s done", project, name)
}

// runScript runs a bash script with optional extra args, returning combined
// stdout+stderr.
func runScript(ctx context.Context, path, workdir string, args ...string) (string, error) {
	cmdArgs := append([]string{path}, args...)
	cmd := exec.CommandContext(ctx, "bash", cmdArgs...)
	cmd.Dir = workdir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}
