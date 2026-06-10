#!/usr/bin/env python3
"""Fast static checks for the local PodScope development loop.

These checks intentionally avoid requiring Go, Docker, kubectl, or minikube so
that the basics can be verified in lightweight CI and editor environments.
"""

from __future__ import annotations

import pathlib
import re
import subprocess
import sys

ROOT = pathlib.Path(__file__).resolve().parents[1]
PODINFO_SELECTOR = "app=podinfo"


def fail(message: str) -> None:
    print(f"FAIL: {message}", file=sys.stderr)
    sys.exit(1)


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def check_make_help() -> None:
    result = subprocess.run(
        ["make", "help"],
        cwd=ROOT,
        text=True,
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        check=False,
    )
    if result.returncode != 0:
        fail(f"make help exited {result.returncode}:\n{result.stdout}")
    for expected in ("PodScope Build Targets:", "Development Workflow:", "Version Management:"):
        if expected not in result.stdout:
            fail(f"make help output missing {expected!r}")


def check_podinfo_selectors() -> None:
    makefile = read("Makefile")
    setup = read("scripts/setup-cluster.sh")
    workload = read("scripts/test-workloads/podinfo.yaml")

    if "app.kubernetes.io/name=podinfo" in makefile or "app.kubernetes.io/name=podinfo" in setup:
        fail("dev-loop commands must use app=podinfo to match the test workload")

    required_snippets = [
        f"./podscope-linux tap -n default -l {PODINFO_SELECTOR}",
        f"kubectl wait --for=condition=ready pod -l {PODINFO_SELECTOR}",
        f"kubectl get pod -l {PODINFO_SELECTOR}",
        f"kubectl get pods -l {PODINFO_SELECTOR}",
    ]
    combined = "\n".join([makefile, setup])
    for snippet in required_snippets:
        if snippet not in combined:
            fail(f"missing dev-loop selector snippet: {snippet}")

    if "app: podinfo" not in workload:
        fail("test workload must label pods with app: podinfo")


def check_readme_transport() -> None:
    readme = read("README.md")
    stale_claims = [
        "Flow events and raw PCAP data are streamed to the Hub via gRPC.",
        "Receives flows from Agents (gRPC)",
    ]
    for claim in stale_claims:
        if claim in readme:
            fail(f"README still contains stale transport claim: {claim}")
    if "HTTP POST" not in readme:
        fail("README should document HTTP POST as the current Agent-to-Hub transport")


def check_go_version_docs() -> None:
    readme = read("README.md")
    if "Go 1.22+" in readme:
        fail("README still advertises Go 1.22+, but go.mod requires Go 1.24")
    if "Go 1.24+" not in readme:
        fail("README should advertise Go 1.24+")


def check_hub_port_help() -> None:
    tap_go = read("pkg/cli/tap.go")
    if "Hub gRPC server" in tap_go:
        fail("--hub-port help should not describe gRPC while HTTP is canonical")
    if not re.search(r'IntVar\(&hubPort, "hub-port", 8080, "[^\"]*HTTP', tap_go):
        fail("--hub-port help should describe the Hub HTTP/API port")


def check_release_images_are_registry_qualified() -> None:
    workflow = read(".github/workflows/release.yml")
    for ldflag in ("DefaultHubImage=dominoramino/podscope:", "DefaultAgentImage=dominoramino/podscope-agent:"):
        if ldflag not in workflow:
            fail(f"release CLI build must embed registry-qualified image default {ldflag!r}")


def main() -> None:
    check_make_help()
    check_podinfo_selectors()
    check_readme_transport()
    check_go_version_docs()
    check_hub_port_help()
    check_release_images_are_registry_qualified()
    print("dev-loop static checks passed")


if __name__ == "__main__":
    main()
