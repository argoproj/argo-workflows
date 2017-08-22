# -*- mode: python -*-

block_cipher = None

import os
SRCROOT = os.path.normpath(SPECPATH + '/../../../')
HOOKSPATH = ['{}/platform/dev/pyinstaller/hooks'.format(SRCROOT)]

a = Analysis([SRCROOT + '/platform/source/tools/axnotification.py'],
             pathex=[SRCROOT + '/common/python'],
             binaries=[],
             datas=[],
             hiddenimports=[],
             hookspath=HOOKSPATH,
             runtime_hooks=[],
             excludes=[],
             win_no_prefer_redirects=False,
             win_private_assemblies=False,
             cipher=block_cipher)

pyz = PYZ(a.pure, a.zipped_data,
             cipher=block_cipher)

exe = EXE(pyz,
          a.scripts,
          a.binaries,
          a.zipfiles,
          a.datas,
          name='axnotification',
          debug=False,
          strip=False,
          upx=True,
          console=True )
