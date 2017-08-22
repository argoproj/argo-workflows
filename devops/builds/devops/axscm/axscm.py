#!/usr/bin/env python3

from ax.util.az_patch import az_patch
az_patch()

from ax.devops.scm.cli import main
if __name__ == '__main__':
    main()
