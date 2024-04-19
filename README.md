# GitOps Tools

A Github Actions to be used in GitOps workflows. Normally this actions is used in workflows inside of your application repository where the source code is stored and the GitOps configuration is stored in a separate repository.

## Usage

```yaml
steps:
  - name: Deploy
    uses: docker://ghcr.io/geode-io/gitops-tools:latest
    env:
      APP_NAME: # Name of the application (optional if app config is provided)
      APP_CONFIG: # Path to the application gitops config (optional if global config is provided)
      GLOBAL_CONFIG: # Path to the global gitops config (optional if app config is provided)
      VALUE: # Value to update the files in the config repository (required)
      GH_TOKEN: # Github PAT with proper permissions (optional if GH_APP_KEY is provided)
      GH_APP_KEY: # Github App private key (optional if GH_TOKEN is provided)
      GH_APP_ID: # Github App ID (optional if GH_TOKEN is provided)
      GH_APP_INSTALLATION_ID: # Github App Installation ID (optional if GH_TOKEN is provided)
      GIT_COMMIT_AUTHOR_NAME: # Name of the commit author (optional)
      GIT_COMMIT_AUTHOR_EMAIL: # Email of the commit author (optional)
      PR_TITLE: # Title of the PR in the config repository (optional)
      PR_BODY: # Body of the PR in the config repository (optional)
```

### Configurations

This action has a configuration file that is used to define how this action should update and deploy the changes to the config repository. Here is the schema of the configuration file:

```yaml
apiVersion: infrastructure.geode.io/v1alpha1
kind: GitOpsConfig
spec:
  configRepo:
    owner: string
    repo: string
    appPathPrefix: string
    app: string
  targetFiles:
    - path: string
      replacer: string
      key: string
      regex:
        pattern: string
        tmpl: string
  deployments:
    - sourceBranch: string
      targetStack: string
      autoDeploy: boolean
```

#### Config Repo

`configRepo` is used to provide information about the config repository where the changes should be pushed.

- `owner`: Owner of the config repository
- `repo`: Name of the config repository
- `appPathPrefix`: Prefix of the path where the configuration files are stored in the config repository
- `app`: Name of the application. It will be used with the combination of `appPathPrefix` to find the path where the configuration files are stored.

#### Target Files

`targetFiles` is used to define the files that should be updated with the provided value.

- `path`: Path of the file in the config repository. It should be relative to the combination of `appPathPrefix` and `app` from the `configRepo`.
- `replacer`: Replacer to be used to update the file. Currently, `yaml` and `regex` are supported.
- `key`: Key to be updated in the file. It is used with the `yaml` replacer.
- `regex`: Regex pattern to be used to update the file. It is used with the `regex` replacer.
- `regex.pattern`: Pattern to be used to find the value to be replaced.
- `regex.tmpl`: Template to be used to replace the value.

#### Deployments

`deployments` is used to define the deployments strategies for the changes.

- `sourceBranch`: Base branch in the config repository where the changes should be pushed.
- `targetStack`: Stack where the changes should be deployed. It is used with the combination of `appPathPrefix` and `app` from the `configRepo`.
- `autoDeploy`: Flag to enable/disable the auto merge of the PR created by this action.

### Example - Mono Repo

Let's say you have a mono repo where you have multiple services source code and you want to update the GitOps configuration after building images and pushing them to the registry. Here is an example of how you can use this action in the mono repo:

Application Mono Repository:

```shell
├── .github
│   └── workflows
│       └── release.yaml
├── gitops-actions.yaml
└── services
    ├── app-1
    │   ├── Cargo.lock
    │   ├── Cargo.toml
    │   ├── Dockerfile
    │   ├── gitops-actions.yaml
    │   └── src
    └── app-2
        ├── Cargo.lock
        ├── Cargo.toml
        ├── Dockerfile
        └── src
```

You can have a global configuration file in the root of the mono repository (or any other path) that defines the configuration for all the services. Here is an example of how the global configuration file can look like:

```yaml
apiVersion: infrastructure.geode.io/v1alpha1
kind: GitOpsConfig
spec:
  configRepo:
    owner: geode-io
    repo: gitops-config
    appPathPrefix: services
  targetFiles:
    - path: config.yaml
      replacer: yaml
      key: tag
    - path: main.tf
      replacer: regex
      regex:
       pattern: '(ref=)([^\"]+)'
       tmpl: '${1}'
  deployments:
    - sourceBranch: main
      targetStack: dev
      autoDeploy: true
    - sourceBranch: main
      targetStack: stage
      autoDeploy: true
    - sourceBranch: main
      targetStack: prod
      autoDeploy: false
```

In the above example, the `targetFiles` defines the files that should be updated with the provided value.
- The first file is a `yaml` file where the `tag` key should be updated with the provided value using the `yaml` replacer.

```yaml
image: 123456789012.dkr.ecr.us-east-1.amazonaws.com/my-company/app-1
tag: 8d9bdc8e05bada480c0011d564910902a812a43a
# other configurations
```
- The second file is a `tf` file where the `ref` value should be updated with the provided value using the `regex` replacer.

```hcl
module "app-1" {
  source = "git@github.com:geode-io/gitar-apps.git//terraform/modules/lambda?ref=8d9bdc8e05bada480c0011d564910902a812a43a"
  ... other configurations
}
```

The `deployments` defines the deployment strategies for the changes. In the above example, pull requests will be created based on the `main` branch in the config repository and auto-merge will be enabled for the `dev` and `stage` stacks.

if you have a separate configuration for a specific service, you can have a configuration file in the service directory. Here is configuration file for `app-1` service which only has `dev` and `stage` stacks:

```yaml
apiVersion: infrastructure.geode.io/v1alpha1
kind: GitOpsConfig
spec:
  configRepo:
    app: app-1
  deployments:
    - sourceBranch: main
      targetStack: dev
      autoDeploy: true
    - sourceBranch: main
      targetStack: stage
      autoDeploy: true
```

> [!NOTE]
> the `spec.configRepo.app` has to be defined at the service level or can be set by the `APP_NAME` environment variable in the action.

GitOps Configuration Repository:
With the above example, the configuration repository will look like this:
```shell
├── .github
│   └── workflows
│       └── required-checks.yaml
└── services
    ├── app-1
    │   ├── dev
    │   │   ├── config.yaml
    │   │   └── main.tf
    │   └── stage
    │       ├── config.yaml
    │       └── main.tf
    └── app-2
        ├── dev
        │   ├── config.yaml
        │   └── main.tf
        ├── prod
        │   ├── config.yaml
        │   └── main.tf
        └── stage
            ├── config.yaml
            └── main.tf
```

> [!TIP]
> It is recommended to have a workflow in the config repository to run some required checks, like `terraform plan`, `ArgoCD Diff`, etc. This action expect the PR has some checks to be passed before auto-merging the PR.


Here is an example of how you can use this action in the mono repository workflow:

```yaml
name: release

on:
  push:
    branches:
      - main
  pull_request:
    branches:
      - main
env:
  AWS_ACCOUNT_ID: 123456789012
  AWS_REGION: us-east-1

jobs:
  affected:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.changed-files.outputs.all_changed_files}}

    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Get changed files
        id: changed-files
        uses: tj-actions/changed-files@v44
        with:
          matrix: true
          dir_names: true
          dir_names_max_depth: 2
          dir_names_exclude_current_dir: true
          files: services/**

  build-docker:
    runs-on: ubuntu-latest
    needs: [affected]
    if: ${{ needs.affected.outputs.matrix != '[]' }}
    strategy:
      fail-fast: false
      matrix:
        service-path: ${{fromJson(needs.affected.outputs.matrix)}}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Login to ECR
        uses: docker/login-action@v3
        if: github.event_name != 'pull_request'
        with:
          registry: ${{ env.AWS_ACCOUNT_ID }}.dkr.ecr.${{ env.AWS_REGION }}.amazonaws.com
          username: ${{ secrets.AWS_ACCESS_KEY_ID }}
          password: ${{ secrets.AWS_SECRET_ACCESS_KEY }}

      - name: Get service name
        id: service-name
        run: |
          echo "service=$(echo ${{ matrix.service-path }} | sed 's/services\///g')" >> "${GITHUB_OUTPUT}"

      - name: Docker meta
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: |
            ${{ env.AWS_ACCOUNT_ID }}.dkr.ecr.${{ env.AWS_REGION }}.amazonaws.com/my-company/${{ steps.service-name.outputs.service }}
          tags: |
            type=raw,value=${{ github.event.pull_request.head.sha || github.sha }}
            type=ref,event=branch
            type=ref,event=pr
          flavor: |
            latest=${{ github.ref == 'refs/heads/main' }}

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Build and push the image
        uses: docker/build-push-action@v5
        with:
          context: ${{ matrix.service-path }}
          platforms: linux/amd64
          push: ${{ github.event_name != 'pull_request' }}
          push: false
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          build-args: |
            APP_NAME=${{ steps.service-name.outputs.service }}

  deploy:
    runs-on: ubuntu-latest
    needs: [affected, build-docker]
    if: ${{ needs.affected.outputs.matrix != '[]' && github.event_name != 'pull_request' }}
    strategy:
      fail-fast: false
      matrix:
        service-path: ${{fromJson(needs.affected.outputs.matrix)}}
    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: get service name drop services/ from path
        id: service-name
        run: |
          echo "service=$(echo ${{ matrix.service-path }} | sed 's/services\///g')" >> "${GITHUB_OUTPUT}"

      - id: create_token
        uses: tibdex/github-app-token@v2
        with:
          app_id: 12345
          private_key: ${{ secrets.PRIVATE_KEY }}

      - name: Deploy
        uses: docker://ghcr.io/geode-io/gitops-tools:latest
        env:
          GH_TOKEN: ${{ steps.create_token.outputs.token }}
          APP_NAME: ${{ steps.service-name.outputs.service }}
          APP_CONFIG: ${{ matrix.service-path }}/gitops-actions.yaml # load the service specific config if exists
          VALUE: ${{ github.sha }}
          GLOBAL_CONFIG: gitops-actions.yaml
          GIT_COMMIT_AUTHOR_NAME: "geode-actions-bot"
```

> [!TIP]
> It is recommended to use a Github App to authenticate with Github API. You can use the `tibdex/github-app-token` action to create a token for the Github App and use it in the action.
