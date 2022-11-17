# chessgames-download
Tool to download PGN games and game collections from chessgames.com

How to use:
.\chessgames-download.exe -url chess-collection-or-game-url -pgn local-pgn-file-to-write-to.pgn

Example for a game collection:


.\chessgames-download.exe -url https://www.chessgames.com/perl/chesscollection?cid=1045049 -pgn .\kings-gambit.pgn

Example for a game: 

.\chessgames-download.exe -url https://www.chessgames.com/perl/chessgame?gid=1003849 -pgn .\spielmann-gruenfeld.pgn
