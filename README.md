Walkthrough and examples on how GitLab 10.7+ and 11.0+ help managing CI when using **monorepos**.

## The problem

As of right now it is only possible to have at most 1 `.gitlab-ci.yml` file per repository. However multiple projects / codebases are housed in one repo when using monorepos which makes it more difficult to use with GitLab CI.

I will show **2 approaches** that tackle this issue.  
Both are based on [variable expressions](https://docs.gitlab.com/ce/ci/variables/README.html#variables-expressions) introduced in GitLab 10.7 (released in late April 2018) that allow you to evaluate variables in `only` and `except` policies.

## Approach 1: Use git commit message to explicitly tell which CI jobs to run
GitLab 11.0 (will be released in late June 2018) makes it possible to [use regex](https://docs.gitlab.com/ce/ci/variables/README.html#supported-syntax) in these variable expressions. Let's look at an example of how to leverage this with monorepos:

In [examples/monorepo](./examples/monorepo) you will find 3 sub-directories. Each one represents a separate codebase in a monorepo. Now let's look at the [.gitlab-ci.yml](./examples/monorepo/.gitlab-ci.yml). The interesting section is the project/codebase - specific configuration:

```
.payment-api: &payment-api
  variables:
    PROJECT_NAME: payment-api
  only:
    variables:
      - $CI_COMMIT_MESSAGE =~ /\[ci job:.*payment-api.*\]/i
      - $RUN_JOBS_ALL == "true"
      - $RUN_JOBS_PAYMENT_API == "true"
```

We define a [hidden CI job](https://docs.gitlab.com/ce/ci/yaml/README.html#hidden-keys-jobs) to use for templating with [YAML anchors](https://docs.gitlab.com/re/ci/yaml/README.html#anchors) and we set project-specific variables that are used in the actual CI jobs, like the project name.  
Now the **most important part**: The `only` policy defines that there are 3 cases when to run this job.

First and foremost:  
If the commit message matches a regex that tells us to run the CI jobs for the payment-api codebase.

Such a commit message could read like this:

```
commit 63e93c0108e5da238d47a20d242c2445922b7204 (local/staging, staging)
Author: John Doe <john@example.com>
Date:   Thu May 24 13:01:25 2018 +0200

    new payment provider: paypal

    - refactored checkout system
    - implemented paypal provider

    [ci job: payment-api customer-dashboard]
```

This will trigger payment-api and customer-dashboard related jobs **only**:

![alt text](./examples/images/jobs_screenshot.png)


If we do not include the other two options here, it will not be possible to (re-)run a pipeline because the commit message regex will never match. When you want to explicitly run the pipeline (e.g. via GitLab web dashboard) for one or more specific codebases, you can set `RUN_JOBS_<CODEBASE_NAME>` to true. If you want to run CI jobs for all codebases, set `RUN_JOBS_ALL` to true.

**Attention**: If you use this approach and push multiple commits to a branch at once, it will only run the jobs specified in the most recent commit message!

As of writing this GitLab 11.0 is not released yet, however you can use the official [Docker image](https://hub.docker.com/r/gitlab/gitlab-ce/tags/) to run a RC or nightly build. The Gitlab runner may have version 10.X when trying this out.

## Approach 2: Determine programmatically which jobs need to run and then trigger CI builds via API with variables

If you don't want to go the git commit message route, we will need to use another way of setting variables dynamically which decide which CI jobs  run.

It is possible to trigger a pipeline via API and passing [variables](https://docs.gitlab.com/ce/ci/triggers/README.html#pass-job-variables-to-a-trigger) along. This gives you a lot of flexibility processing which variables to set.

For monorepos it is important to know/determine which files changed. If only files in the payment-api codebase changed, we only need to run the CI jobs associated with this project.

If you are not already triggering your builds via API and instead want to continue to use the "traditional" CI triggering mechanism your GitLab CI flow will currently look something like this:

```
git push
==> pipeline gets triggered implicitly by GitLab
==> GitLab evaluates CI file and starts pipeline
==> CI pipeline runs
```

One approach to preserve this while also adding file change detection for monorepos, we will need to add a custom component, which acts as an endpoint to a [GitLab webhook](https://docs.gitlab.com/ce/user/project/integrations/webhooks.html) and will then trigger the pipeline explicitly via API, so our flow now looks like:

```
git push
==> GitLab Webhook gets triggered
==> Push event payload including which files changed is sent to our components endpoint
==> component determines which CI jobs need to run and sets variables accordingly
==> Trigger pipeline via API including trigger variables
==> GitLab evaluates CI file and starts pipeline
==> CI pipeline runs
```
