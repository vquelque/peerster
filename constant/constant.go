package constant

const ChannelSize = 50
const DefaultHopLimit = 10 //for private messages routing (in Hops)
const AckTimeout = 10      //in seconds

const ChunkSize = 8192 //in bytes
const FileTempDirectory = "./_SharedFiles/"
const FileOutDirectory = "./_Downloads/"
const MaxChunkDownloadTries = 10
const Timeout = 5 //in seconds

const SearchRequestTimeout = 500 //in ms
const DefaultSearchBudget = 2
const SearchRequestResendTimer = 1
const MaxBudget = 32
const SearchMatchThreshold = 2
const SearchRequestMaxRetries = 7

const DefaultStubbornTimeout = 5
