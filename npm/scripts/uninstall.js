const fs = require('fs');
const path = require('path');
const os = require('os');

function uninstall() {
    console.log('Uninstalling Myrai...\n');
    
    const binDir = path.join(__dirname, '..', 'bin');
    
    if (fs.existsSync(binDir)) {
        const files = fs.readdirSync(binDir);
        files.forEach(file => {
            if (file.startsWith('myrai')) {
                const filePath = path.join(binDir, file);
                fs.unlinkSync(filePath);
                console.log(`Removed: ${filePath}`);
            }
        });
    }
    
    console.log('\n✓ Myrai uninstalled');
}

uninstall();
