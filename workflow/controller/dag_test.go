package controller

var diamondDAG = `
targets:
- name: A
  template: whalesay
- name: B
  dependencies: [A]
  template: whalesay
- name: C
  dependencies: [A]
  template: whalesay
- name: D
  dependencies: [B, C]
  template: whalesay
`
