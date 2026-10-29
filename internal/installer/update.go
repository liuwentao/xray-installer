package installer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

type UpdateRequest struct {
	Version string
	DryRun  bool
}

var xrayVersionRE = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z]+(?:[.-][0-9A-Za-z]+)*)?$`)

func normalizeXrayVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version == "" {
		return "", nil
	}
	if !xrayVersionRE.MatchString(version) {
		return "", fmt.Errorf("invalid Xray version %q; expected vX.Y.Z", version)
	}
	return "v" + strings.TrimPrefix(version, "v"), nil
}

func (i *Installer) Update(ctx context.Context, req UpdateRequest) error {
	version, err := normalizeXrayVersion(req.Version)
	if err != nil {
		return err
	}
	if runtime.GOOS != "linux" {
		return errors.New("only Linux is supported")
	}
	if !req.DryRun && os.Geteuid() != 0 {
		return errors.New("please run update as root")
	}
	if !pathExists(xrayBinary) || !pathExists(xrayConfigPath) || !pathExists("/etc/systemd/system/xray.service") {
		return errors.New("Xray installation is incomplete; run xray-installer to install first")
	}
	if _, err := exec.LookPath("systemctl"); err != nil {
		return errors.New("systemctl not found; systemd is required")
	}
	if !pathExists("/run/systemd/system") {
		return errors.New("systemd does not appear to be active")
	}
	osID, osLike, err := readOSRelease()
	if err != nil {
		return err
	}
	if !supportsDistro(osID, osLike) {
		return fmt.Errorf("unsupported distro %q; only Debian/Ubuntu/Arch Linux are supported", osID)
	}
	target := version
	if target == "" {
		target = "最新稳定版"
	}
	if req.DryRun {
		fmt.Fprintf(i.stdout, "Dry run：将更新 Xray 到 %s，保留现有配置，校验配置后重启并检查服务。未下载或修改任何文件。\n", target)
		return nil
	}
	i.step("更新 Xray 到 " + target)
	// Keep the existing service settings and validate before restarting.
	options := []string{"--no-update-service", "--manually-restart-later"}
	if version != "" {
		options = append(options, "--version", version)
	}
	if err := i.installXray(ctx, options...); err != nil {
		return err
	}
	if output, err := commandOutput(ctx, xrayBinary, "run", "-test", "-config", xrayConfigPath); err != nil {
		return fmt.Errorf("updated Xray cannot use the existing config: %w\n%s", err, strings.TrimSpace(output))
	}
	i.step("重启并验证 Xray 服务")
	if err := i.enableAndRestartXray(ctx); err != nil {
		return err
	}
	fmt.Fprintln(i.stdout, "\nXray 更新完成，现有节点和客户端配置已保留。")
	return nil
}
