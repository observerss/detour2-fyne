; The name of the installer
Name "Detour2"

!define PRODUCT_VERSION "0.2.0"

; To change from default installer icon:
;Icon "Icon.png"

; The setup filename
OutFile "dist\Detour_Setup_v0.2.0.exe"

; The default installation directory
InstallDir $LOCALAPPDATA\Programs\Detour2

; Registry key to check for directory (so if you install again, it will 
; overwrite the old one automatically)
InstallDirRegKey HKCU "Software\Detour2" "Install_Dir"

RequestExecutionLevel user

VIProductVersion "0.2.0.0"
VIAddVersionKey "ProductName" "Detour2"
VIAddVersionKey "CompanyName" "Detour2"
VIAddVersionKey "FileDescription" "Detour2 local proxy client"
VIAddVersionKey "FileVersion" "${PRODUCT_VERSION}"
VIAddVersionKey "ProductVersion" "${PRODUCT_VERSION}"

;--------------------------------

; Pages

Page components
Page directory
Page instfiles

UninstPage uninstConfirm
UninstPage instfiles

;--------------------------------

; The stuff to install
Section "Detour2 (required)"

  SectionIn RO
  
  ; Set output path to the installation directory.
  SetOutPath $INSTDIR
  
  ; Put file there (you can add more File lines too)
  File dist\Detour2.exe
  ; Wildcards are allowed:
  ; File *.dll
  ; To add a folder named MYFOLDER and all files in it recursively, use this EXACT syntax:
  ; File /r MYFOLDER\*.*
  ; See: https://nsis.sourceforge.io/Reference/File
  ; MAKE SURE YOU PUT ALL THE FILES HERE IN THE UNINSTALLER TOO
  
  ; Write the installation path into the registry
  WriteRegStr HKCU SOFTWARE\Detour2 "Install_Dir" "$INSTDIR"
  
  ; Write the uninstall keys for Windows
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "DisplayName" "Detour2"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "Publisher" "Detour2"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "UninstallString" '"$INSTDIR\uninstall.exe"'
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "NoModify" 1
  WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2" "NoRepair" 1
  WriteUninstaller "$INSTDIR\uninstall.exe"
  
SectionEnd

; Optional section (can be disabled by the user)
Section "Start Menu Shortcuts (required)"
  SectionIn RO

  CreateDirectory "$SMPROGRAMS\Detour2"
  CreateShortcut "$SMPROGRAMS\Detour2\Uninstall.lnk" "$INSTDIR\uninstall.exe" "" "$INSTDIR\uninstall.exe" 0
  CreateShortcut "$SMPROGRAMS\Detour2\Detour2.lnk" "$INSTDIR\Detour2.exe" "" "$INSTDIR\Detour2.exe" 0
  
SectionEnd

;--------------------------------

; Uninstaller

Section "Uninstall"
  
  ; Remove registry keys
  DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\Detour2"
  DeleteRegKey HKCU SOFTWARE\Detour2

  ; Remove files and uninstaller
  ; MAKE SURE NOT TO USE A WILDCARD. IF A
  ; USER CHOOSES A STUPID INSTALL DIRECTORY,
  ; YOU'LL WIPE OUT OTHER FILES TOO
  Delete $INSTDIR\Detour2.exe
  Delete $INSTDIR\uninstall.exe

  ; Remove shortcuts, if any
  Delete "$SMPROGRAMS\Detour2\*.*"

  ; Remove directories used (only deletes empty dirs)
  RMDir "$SMPROGRAMS\Detour2"
  RMDir "$INSTDIR"

SectionEnd