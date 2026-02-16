#!/usr/bin/env node

const path = require('path');
const binaryPath = path.join(__dirname, '..', 'bin', 'myrai-' + process.platform + '-' + process.arch + (process.platform === 'win32' ? '.exe' : ''));

require('child_process').spawn(binaryPath, process.argv.slice(2), { 
    stdio: 'inherit',
    env: process.env
}).on('exit', process.exit);
