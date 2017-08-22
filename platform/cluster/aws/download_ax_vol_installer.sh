# Download the installer
AX_VOL_LOCAL_PATH=/tmp/ax_vol_plugin.tar.gz

wget https://s3-us-west-1.amazonaws.com/ax-public/ax_vol_plugin/installer/AX_VOL_INSTALLER_VERSION/ax_vol_plugin.tar.gz -O ${AX_VOL_LOCAL_PATH}

# Untar and run  the installer.
tar -zxvf ${AX_VOL_LOCAL_PATH}
ORIG_WD=${PWD}
cd ax_vol_plugin
chmod u+x vol_plugin_installer.sh
./vol_plugin_installer.sh
cd ${ORIG_WD}
echo "AX: Volume plugin installer ran successfully"
