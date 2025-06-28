# GitHub Agile Workflow

This directory contains the complete agile workflow configuration for the mkunion project, designed to support collaboration between engineers, AI assistants, and product owners.

## Overview

The workflow implements a comprehensive agile process with the following key features:

- **AI Integration**: Support for AI-generated and AI-refined issues
- **Automated Triage**: Issues automatically flow through appropriate states
- **Capacity Management**: Prevents team overload with WIP limits
- **Sprint Automation**: Automated sprint planning and tracking
- **Quality Gates**: Ensures issues and PRs meet quality standards
- **SLA Monitoring**: Tracks response times and escalates violations
- **Comprehensive Metrics**: Velocity, burndown, and team performance tracking

## Workflow States

1. **Inbox** → New issues land here for initial triage
2. **Human Review** → AI-generated issues requiring validation
3. **AI Refinement** → Issues sent back to AI for more details
4. **Backlog** → Approved issues ready for prioritization
5. **TODO** → Prioritized for current/next sprint
6. **In Progress** → Actively being worked on (max 2 per engineer)
7. **Code Review** → PR submitted and under review
8. **Acceptance** → Deployed and awaiting verification
9. **Done** → Completed and verified (auto-cleaned biweekly)

## Key Workflows

### Issue Management
- `issue-inbox.yml` - Automatic triage and labeling of new issues
- `issue-lifecycle.yml` - State transition management
- `issue-validator.yml` - Quality validation for issues

### PR Management
- `pr-lifecycle.yml` - PR automation and review tracking
- `pr-readiness.yml` - PR quality checks and readiness scoring

### Capacity Management
- `capacity-check.yml` - Monitor and enforce WIP limits
- `workload-balancer.yml` - Automatic task distribution

### Sprint Management
- `sprint-starter.yml` - Sprint planning and kickoff
- `sprint-metrics.yml` - Daily metrics collection
- `burndown-generator.yml` - Burndown chart generation
- `velocity-tracker.yml` - Team velocity tracking

### Maintenance
- `stale-backlog.yml` - Mark and clean stale backlog items
- `cleanup-done.yml` - Archive completed items biweekly

### Communication
- `daily-digest.yml` - Daily team summary
- `blocker-alerts.yml` - Escalation for blocked issues
- `sla-monitor.yml` - SLA tracking and violations

### Quality Assurance
- `acceptance-checklist.yml` - Acceptance criteria verification

## Configuration Files

- `project.yml` - Project board structure and automation
- `labels.yml` - Complete label taxonomy
- `CODEOWNERS` - Code review assignments
- `validation-rules/` - Quality standards for issues and PRs
- `notification-rules/` - Escalation matrix and alerts

## Usage

### For Engineers

1. **Taking Work**: Move issues from TODO to In Progress (max 2 at a time)
2. **Blocking Issues**: Add `blocked` label with explanation
3. **Code Review**: PRs automatically tracked, aim for <4 hour response
4. **Completion**: Issues auto-move through states based on PR status

### For AI Assistants

1. **Creating Issues**: Use `ai-generated.yml` template
2. **Refinement**: Watch for `needs-ai-refinement` label
3. **Code Review**: Can be requested with `ai-review-requested` label

### For Product Owners

1. **Review Queue**: Check "Human Review" column daily
2. **Prioritization**: Move approved issues to TODO
3. **Acceptance**: Verify completed work in "Acceptance" column
4. **Sprint Planning**: Review metrics and velocity reports

### For Tech Leads

1. **Escalations**: Monitor blocker alerts and SLA violations
2. **Capacity**: Review team utilization reports
3. **Quality**: Ensure code review standards are met
4. **Architecture**: Review large PRs and system changes

## Metrics and Reports

The system automatically generates:

- **Daily Digest**: Team activity summary
- **Sprint Reports**: Burndown, velocity, completion rates
- **Capacity Reports**: Team utilization and workload
- **SLA Reports**: Response time violations
- **Stale Item Reports**: Aging backlog items

## Environment Variables

Configure these in your repository settings:

```bash
TEAM_SIZE=5
TEAM_MEMBERS=user1,user2,user3,user4,user5
TECH_LEAD_USERNAME=tech-lead-github-username
ENGINEERING_MANAGER_USERNAME=manager-github-username
PRODUCT_OWNER_USERNAME=po-github-username
SCRUM_MASTER_USERNAME=sm-github-username
```

## Getting Started

1. **Enable GitHub Actions** in your repository
2. **Create a Project Board** and link it to the repository
3. **Apply Labels** by running: `gh label create -F .github/labels.yml`
4. **Configure Team** by setting environment variables
5. **Start Creating Issues** using the provided templates

## Customization

All workflows are designed to be customizable:

- Adjust SLA times in `sla-monitor.yml`
- Modify WIP limits in `capacity-check.yml`
- Change sprint duration in `sprint-starter.yml`
- Update quality rules in `validation-rules/`

## Troubleshooting

- **Workflows not running**: Check Actions are enabled and permissions are set
- **Labels missing**: Run label creation command above
- **Assignments failing**: Verify usernames in environment variables
- **Metrics missing**: Ensure scheduled workflows have run at least once

## Contributing

When modifying workflows:

1. Test changes in a fork first
2. Document any new environment variables
3. Update this README with significant changes
4. Ensure backwards compatibility

## Support

For issues or questions about the agile workflow, create an issue with the `workflow` label.