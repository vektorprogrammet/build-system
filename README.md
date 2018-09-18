# Build System
Automatically deploys PRs to [branchname.staging.vektorprogrammet.no](https://branchname.staging.vektorprogrammet.no)

## CLI documentation
### To deploy a new branch
```bash
./staging-server deploy-branch [branch name]
```

### To stop a hosted server
```bash
./staging-server deploy-branch --delete [branch name]
./staging-server deploy-branch -d [branch name] #shorthand
```

### To list all servers
```bash
./staging-server list-servers
./staging-server ls #shorthand
```
