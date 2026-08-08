Output assets/demo.gif
Set FontSize 13
Set Width 1100
Set Height 560
Set Padding 18

# Build the binary from the checked-out repo (vhs runs at repo root).
Type "go build -o /tmp/dropdiff ."
Enter
Sleep 2s

# Isolate HOME so the demo does not touch the runner's real ~/.dropdiff.
Type "export HOME=/tmp/dropdiff-demo"
Enter
Type "rm -rf $HOME && mkdir -p $HOME/demo && cd $HOME/demo"
Enter

# Wire a GLM-4.6 endpoint with implicit cache (the field the GLM-5.3 catch hits).
Type "printf 'MODEL=glm-4.6\\nOPENAI_API_BASE=https://open.bigmodel.cn\\nCACHE_MODE=implicit\\nMAX_CONTEXT=128000\\n' > .env"
Enter
Sleep 500ms

# m1: init snapshots the wired config.
Type "/tmp/dropdiff init --cwd ."
Enter
Sleep 1500ms

# m3: check prints the GLM-5.3 cache-price catch (0.28 -> 0.14), green.
Type "/tmp/dropdiff check"
Enter
Sleep 2500ms

# m2: list shows the curated registry (>=8 drops with sources).
Type "/tmp/dropdiff list"
Enter
Sleep 2500ms

# version
Type "/tmp/dropdiff --version"
Enter
Sleep 1s
