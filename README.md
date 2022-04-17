# ZFIT - Zwift FIT file analyzer

`zfit` is a CLI and library that makes it easy to analyze Zwift .FIT files. 

- [ZFIT - Zwift FIT file analyzer](#zfit---zwift-fit-file-analyzer)
  - [Features](#features)
  - [Installation](#installation)
  - [Usage](#usage)
## Features
- Create database snapshots with the `build` command
- Load database snapshots to a target database
- Specify multiple target configurations to represent your workflow
- Sensitive data in configuration file is encrypted

## Installation

`brew install joescharf/tap/zfit`

## Usage

`zfit process <zwift_fit_file> [--kg 72]`
