# The image workline's CI templates run in: the engine, and the tools its
# roles use, at fixed versions, so a job starts at once instead of building
# them (docs/BACKLOG.md, "Install without Go in CI"). No AI agent inside:
# Claude Code is installed by the job that needs it, with its token.
# Built by .github/workflows/release.yml from the release's linux binary.
FROM node:22-bookworm-slim
ARG GITLEAKS=8.30.1
ARG GLAB=1.120.0
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates curl git jq \
 && rm -rf /var/lib/apt/lists/* \
 && cd /tmp \
 && curl -fsSLO "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS}/gitleaks_${GITLEAKS}_linux_x64.tar.gz" \
 && curl -fsSLO "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS}/gitleaks_${GITLEAKS}_checksums.txt" \
 && grep " gitleaks_${GITLEAKS}_linux_x64.tar.gz\$" "gitleaks_${GITLEAKS}_checksums.txt" | sha256sum -c - \
 && tar -xzf "gitleaks_${GITLEAKS}_linux_x64.tar.gz" -C /usr/local/bin gitleaks \
 && curl -fsSLO "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB}/downloads/glab_${GLAB}_linux_amd64.tar.gz" \
 && curl -fsSLO "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB}/downloads/checksums.txt" \
 && grep " glab_${GLAB}_linux_amd64.tar.gz\$" checksums.txt | sha256sum -c - \
 && tar -xzf "glab_${GLAB}_linux_amd64.tar.gz" -C /usr/local bin/glab \
 && rm -f /tmp/*
COPY workline /usr/local/bin/workline
RUN workline version && gitleaks version && glab --version
