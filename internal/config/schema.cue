// Closed definitions reject unknown fields at every mapping level.
#Mapping: {
 repo: string & !=""
 root: string & !=""
}
#Group: {mappings?: [...#Mapping]}
#Config: {
 github?: #Group
 local?: #Group
}
