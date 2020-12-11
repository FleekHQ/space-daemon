package installer

import (
	"context"
	"os/exec"

	"github.com/pkg/errors"

	"github.com/FleekHQ/space-daemon/log"
	"github.com/keybase/go-kext"
)

type State int64

const (
	Default State = iota
	Downloading
	Installing
	Error
)

type macFuseInstaller struct {
	state State
}

func NewFuseInstaller() *macFuseInstaller {
	return &macFuseInstaller{
		state: Default,
	}
}

func (d *macFuseInstaller) IsInstalled(ctx context.Context) (bool, error) {
	info, err := kext.LoadInfo("com.github.osxfuse.filesystems.osxfuse")
	if err != nil {
		log.Error("unable to determine state of extension", err)
		return false, err
	}

	return info != nil, nil
}

// Install assumes that the Fuse .pkg installer exists in a particular directory
func (d *macFuseInstaller) Install(ctx context.Context, args map[string]interface{}) error {
	// ideally, this should download the fuse pkg and call the installer

	// first starting with providing a path for it, will change to download as this is a security risk
	d.state = Installing
	path, ok := args["path"].(string)
	if !ok {
		return errors.New("'path' is missing from install arguments")
	}

	installerPath, err := exec.LookPath("installer")
	if err != nil {
		return errors.Wrap(err, "pkg installer not present")
	}

	cmd := exec.Command(installerPath, "-pkg", path, "-target", "/")
	out, err := cmd.CombinedOutput()
	log.Debug("Install command output: " + string(out))
	if err != nil {
		return err
	}

	// load the kernel extension
	return d.loadKernel()
}

func (d *macFuseInstaller) loadKernel() error {
	log.Debug("Loading OSXFUSE Kernel")
	cmd := exec.Command("/Library/Filesystems/osxfuse.fs/Contents/Resources/load_osxfuse")
	output, err := cmd.CombinedOutput()
	log.Debug("Kernel Loading Output: " + string(output))
	return err
}
