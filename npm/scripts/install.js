const https = require('https');
const fs = require('fs');
const path = require('path');
const os = require('os');

const REPO = 'gmsas95/goclawde-cli';
const BINARY_NAME = 'myrai';

function getPlatform() {
    switch (process.platform) {
        case 'darwin': return 'darwin';
        case 'linux': return 'linux';
        case 'win32': return 'windows';
        default: return 'linux';
    }
}

function getArch() {
    switch (process.arch) {
        case 'x64': return 'amd64';
        case 'arm64': return 'arm64';
        case 'arm': return 'arm';
        default: return 'amd64';
    }
}

function getLatestVersion() {
    return new Promise((resolve, reject) => {
        https.get(`https://api.github.com/repos/${REPO}/releases/latest`, {
            headers: { 'User-Agent': 'myrai-installer' }
        }, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                try {
                    const release = JSON.parse(data);
                    resolve(release.tag_name || 'v0.1.0');
                } catch (e) {
                    resolve('v0.1.0');
                }
            });
        }).on('error', reject);
    });
}

function downloadFile(url, dest) {
    return new Promise((resolve, reject) => {
        const file = fs.createWriteStream(dest);
        
        const request = (url) => {
            https.get(url, (res) => {
                if (res.statusCode === 302 || res.statusCode === 301) {
                    request(res.headers.location);
                    return;
                }
                
                if (res.statusCode !== 200) {
                    reject(new Error(`Failed to download: ${res.statusCode}`));
                    return;
                }
                
                res.pipe(file);
                file.on('finish', () => {
                    file.close();
                    resolve();
                });
            }).on('error', (err) => {
                fs.unlink(dest, () => {});
                reject(err);
            });
        };
        
        request(url);
    });
}

async function install() {
    console.log('🤖 Installing Myrai...\n');
    
    const platform = getPlatform();
    const arch = getArch();
    const ext = platform === 'windows' ? '.exe' : '';
    
    console.log(`Platform: ${platform}/${arch}`);
    
    let version;
    try {
        version = await getLatestVersion();
        console.log(`Version: ${version}`);
    } catch (e) {
        console.log('Version: latest (could not determine)');
        version = 'latest';
    }
    
    const filename = `${BINARY_NAME}-${platform}-${arch}${ext}`;
    const downloadUrl = `https://github.com/${REPO}/releases/download/${version}/${filename}`;
    
    const binDir = path.join(__dirname, '..', 'bin');
    if (!fs.existsSync(binDir)) {
        fs.mkdirSync(binDir, { recursive: true });
    }
    
    const binaryPath = path.join(binDir, `myrai-${platform}-${arch}${ext}`);
    
    console.log(`\nDownloading from: ${downloadUrl}`);
    
    try {
        await downloadFile(downloadUrl, binaryPath);
        console.log('✓ Download complete');
        
        // Make executable on Unix
        if (platform !== 'windows') {
            fs.chmodSync(binaryPath, 0o755);
        }
        
        // Also create a generic symlink/copy
        const genericPath = path.join(binDir, BINARY_NAME + ext);
        fs.copyFileSync(binaryPath, genericPath);
        if (platform !== 'windows') {
            fs.chmodSync(genericPath, 0o755);
        }
        
        console.log('\n✅ Myrai installed successfully!\n');
        console.log('Run: npx myrai --help');
        console.log('Or: myrai --help (if installed globally)\n');
        
    } catch (error) {
        console.error('\n❌ Installation failed:', error.message);
        console.error('\nYou can build from source:');
        console.error('  git clone https://github.com/' + REPO + '.git');
        console.error('  cd goclawde-cli');
        console.error('  make build\n');
        process.exit(1);
    }
}

install();
