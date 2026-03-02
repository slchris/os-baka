
import React from 'react';
import { Book, Server, Shield, Code, Cpu, RotateCcw } from 'lucide-react';

export const Documentation: React.FC = () => {
  return (
    <div className="space-y-8 max-w-5xl animate-fade-in">
      <div>
        <h2 className="text-2xl font-bold text-gray-900 dark:text-white">System Documentation</h2>
        <p className="text-gray-500 dark:text-gray-400">Technical guides for OS-Baka deployment architecture.</p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* Navigation / TOC */}
        <div className="space-y-4">
          <h3 className="font-semibold text-gray-900 dark:text-white">On this page</h3>
          <ul className="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <li className="hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer">Architecture Overview</li>
            <li className="hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer">Full Disk Encryption (LUKS2)</li>
            <li className="hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer">Network Boot (PXE/DNSMasq)</li>
            <li className="hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer">WebShell Integration</li>
            <li className="hover:text-blue-600 dark:hover:text-blue-400 cursor-pointer">Restoration Procedures</li>
          </ul>

          <div className="bg-blue-50 dark:bg-blue-900/20 p-4 rounded-xl mt-8">
            <p className="text-xs text-blue-800 dark:text-blue-300 font-medium">
              Need API Access? <br />
              Check the <code>/api-docs</code> endpoint on the backend service container.
            </p>
          </div>
        </div>

        {/* Main Content */}
        <div className="lg:col-span-2 space-y-8">

          <section className="bg-white dark:bg-gray-800 p-6 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg"><Server className="w-5 h-5 text-gray-700 dark:text-gray-300" /></div>
              <h3 className="text-lg font-bold text-gray-900 dark:text-white">Architecture</h3>
            </div>
            <p className="text-gray-600 dark:text-gray-300 leading-relaxed mb-4">
              OS-Baka uses a controller-agent architecture. The central management node runs a PXE boot server (Dnsmasq) and serves Preseed files via HTTP.
            </p>
            <div className="bg-gray-900 rounded-lg p-4 overflow-x-auto">
              <pre className="text-xs text-green-400 font-mono">
                {`[Mgmt Node]
  |-- DHCP/TFTP (Dnsmasq)
  |-- HTTP Server (Nginx)
  |-- API Backend (Node.js)
  |-- Database (Postgres)
  
[Target Node]
  |-- PXE Boot -> Fetch Kernel
  |-- Auto-Install (Preseed)
  |-- LUKS Enrollment (TPM2/USB)`}
              </pre>
            </div>
          </section>

          <section className="bg-white dark:bg-gray-800 p-6 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg"><Shield className="w-5 h-5 text-gray-700 dark:text-gray-300" /></div>
              <h3 className="text-lg font-bold text-gray-900 dark:text-white">Security & Encryption</h3>
            </div>
            <p className="text-gray-600 dark:text-gray-300 leading-relaxed mb-4">
              We utilize <strong>LUKS2</strong> with AES-XTS-PLAIN64 cipher. Keys are managed hierarchically:
            </p>
            <ul className="list-disc list-inside space-y-2 text-gray-600 dark:text-gray-300 ml-2">
              <li><strong>Slot 0:</strong> TPM2 bound key (Auto-unlock if PCRs match).</li>
              <li><strong>Slot 1:</strong> Recovery passphrase (stored in Vault).</li>
              <li><strong>Slot 2:</strong> USB Keyfile (Optional, physical token).</li>
            </ul>
          </section>

          <section className="bg-white dark:bg-gray-800 p-6 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg"><Code className="w-5 h-5 text-gray-700 dark:text-gray-300" /></div>
              <h3 className="text-lg font-bold text-gray-900 dark:text-white">Automation</h3>
            </div>
            <p className="text-gray-600 dark:text-gray-300 leading-relaxed">
              CI/CD pipelines via GitHub Actions automatically build the installer images. When you import a CSV, the system maps MAC addresses to static IPs in <code>/etc/dnsmasq.d/hosts.conf</code>.
            </p>
          </section>

          <section className="bg-white dark:bg-gray-800 p-6 rounded-2xl border border-gray-200 dark:border-gray-700 shadow-sm">
            <div className="flex items-center gap-3 mb-4">
              <div className="p-2 bg-gray-100 dark:bg-gray-700 rounded-lg"><RotateCcw className="w-5 h-5 text-gray-700 dark:text-gray-300" /></div>
              <h3 className="text-lg font-bold text-gray-900 dark:text-white">Restoration Procedures</h3>
            </div>
            <div className="space-y-4 text-gray-600 dark:text-gray-300">
              <p className="leading-relaxed">
                In the event of a TPM failure, mainboard replacement, or corrupted bootloader, the automatic unlocking of the LUKS container will fail. Follow these steps to restore access:
              </p>

              <div className="bg-orange-50 dark:bg-orange-900/20 border-l-4 border-orange-500 p-4">
                <p className="text-sm text-orange-700 dark:text-orange-300 font-medium">
                  <strong>Critical:</strong> Ensure you have access to the 'Key Vault' page before proceeding. You will need the 'Admin Recovery Key' (Slot 1).
                </p>
              </div>

              <ol className="list-decimal list-inside space-y-3 ml-2">
                <li>Boot the node. When prompted for the LUKS passphrase, press <code>ESC</code> to see the prompt.</li>
                <li>Navigate to the <strong>Key Vault</strong> in OS-Baka dashboard.</li>
                <li>Find the target node and click <strong>Generate Passphrase</strong> (this retrieves the stored recovery key).</li>
                <li>Enter the passphrase manually into the console.</li>
                <li>Once booted, re-enroll the TPM using the <code>cryptsetup</code> utility or via the OS-Baka agent:</li>
              </ol>

              <div className="bg-gray-900 rounded-lg p-4">
                <code className="text-xs text-gray-300 font-mono">
                  # Re-enroll TPM2<br />
                  sudo systemd-cryptenroll --tpm2-device=auto --tpm2-pcrs=0,7 /dev/sda3<br />
                  <br />
                  # Verify Slots<br />
                  sudo cryptsetup luksDump /dev/sda3
                </code>
              </div>
            </div>
          </section>

        </div>
      </div>
    </div>
  );
};
