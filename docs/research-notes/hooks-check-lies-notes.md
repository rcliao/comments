# Hooks Check Lies: research notes

Command outputs captured 2026-10-08 while shaping hooks-check-lies.md.

```
$ git config --show-origin --get-all core.hooksPath
file:/Users/rcliao/src/rcliao/comments/.git/config	/Users/rcliao/src/rcliao/comments/.githooks
$ ls -l /Users/rcliao/src/rcliao/comments/.githooks
-rwxr-xr-x@ 1 rcliao  staff   711 Aug 10 08:01 pre-commit
-rwxr-xr-x@ 1 rcliao  staff  1150 Aug 10 08:01 pre-push
$ git rev-parse --git-path hooks        # in the agent worktree
/Users/rcliao/src/rcliao/comments/.githooks
$ git -C scratch/sub rev-parse --git-path hooks   # core.hooksPath=.githooks
../.githooks
$ git -C scratch rev-parse --git-path hooks       # core.hooksPath=/abs/x
/abs/x
```
