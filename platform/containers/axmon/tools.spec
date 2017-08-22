# -*- mode: python -*-

block_cipher = None

import os
SRCROOT = os.path.normpath(SPECPATH + '/../../../')
HOOKSPATH = ['{}/platform/dev/pyinstaller/hooks'.format(SRCROOT)]

tools = ['axmon', 'axtool']

a = Analysis(['{}/platform/source/tools/{}.py'.format(SRCROOT, tool) for tool in tools],
             pathex=['{}/common/python'.format(SRCROOT)],
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

for script in [s for s in a.scripts if s[0] in tools]:
    exe = EXE(pyz,
              [script],
              a.binaries,
              a.zipfiles,
              a.datas,
              name=script[0],
              debug=False,
              strip=False,
              upx=True,
              console=True )
