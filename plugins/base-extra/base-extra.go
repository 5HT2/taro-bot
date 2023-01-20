package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/5HT2/taro-bot/bot"
	"github.com/5HT2/taro-bot/cmd"
	"github.com/5HT2/taro-bot/plugins"
	"github.com/5HT2/taro-bot/util"
	"github.com/5HT2C/http-bash-requests/httpBashRequests"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

func InitPlugin(_ *plugins.PluginInit) *plugins.Plugin {
	return &plugins.Plugin{
		Name:        "Taro Base Extra",
		Description: "The extra commands as included as part of the bot",
		Version:     "1.0.0",
		Commands: []bot.CommandInfo{{
			Fn:          ChannelCommand,
			FnName:      "ChannelCommand",
			Name:        "channel",
			Aliases:     []string{"c"},
			Description: "Manage channels",
			GuildOnly:   true,
		}, {
			Fn:          PermissionCommand,
			FnName:      "PermissionCommand",
			Name:        "permission",
			Aliases:     []string{"perm"},
			Description: "Manage user permissions",
			GuildOnly:   true,
		}, {
			Fn:          ProfilePicCommand,
			FnName:      "ProfilePicCommand",
			Name:        "profilepic",
			Aliases:     []string{"pfp", "avatar"},
			Description: "Get the profile picture of someone",
		}, {
			Fn:          SudoCommand,
			FnName:      "SudoCommand",
			Name:        "sudo",
			Aliases:     []string{"#", "su"},
			Description: "Operator-only commands",
		}},
		Responses: []bot.ResponseInfo{{
			Fn:           BashResponse,
			Regexes:      []string{"."},
			MatchMin:     1,
			LockChannels: []int64{bot.C.OperatorChannel},
		}},
	}
}

func BashResponse(r bot.Response) {
	if !r.E.GuildID.IsValid() {
		return
	}

	cmd.CommandHandlerWithCommand(r.E, "#", strings.Split(r.E.Message.Content, " "))
}

func ChannelCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)
	arg2, _ := cmd.ParseStringArg(c.Args, 2, true)

	defaultResponse := func() error {
		_, err := cmd.SendEmbed(c.E,
			"Channel",
			"Available arguments are:\n- `archive`\n- `archive role|category [role id|category id]`\n- `slow [seconds]`",
			bot.DefaultColor)
		return err
	}

	switch arg1 {
	case "archive":
		if err := cmd.HasPermission(c, cmd.PermChannels); err != nil {
			return err
		}

		switch arg2 {
		case "role":
			var errCtx error
			role, err := cmd.ParseInt64Arg(c.Args, 3)
			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {

				if err != nil {
					set := fmt.Sprintf("currently set to <@&%v>!", g.ArchiveRole)
					setColor := bot.DefaultColor
					if g.ArchiveRole == 0 {
						set = "not set."
						setColor = bot.WarnColor
					}
					_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Role", set, setColor)
				} else {
					g.ArchiveRole = role
					_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Role", fmt.Sprintf("Set to <@&%v>!", role), bot.SuccessColor)
				}
				return g, "ChannelCommand: set guild role"
			})
			return errCtx
		case "category":
			var errCtx error
			category, err := cmd.ParseInt64Arg(c.Args, 3)
			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				if err != nil {
					set := fmt.Sprintf("currently set to <#%v>!", g.ArchiveCategory)
					setColor := bot.DefaultColor
					if g.ArchiveCategory == 0 {
						set = "not set."
						setColor = bot.WarnColor
					}
					_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Category", set, setColor)
				} else {
					g.ArchiveCategory = category
					_, errCtx = cmd.SendEmbed(c.E, "Channel Archive Category", fmt.Sprintf("Set to <#%v>!", category), bot.SuccessColor)
				}
				return g, "ChannelCommand: set guild role"
			})
			return errCtx
		case "":
			var err error
			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				if g.ArchiveCategory == 0 {
					err = bot.GenericError(c.FnName, "getting archive category", "`archive_category` not set, use `archive category [category id]`")
				}
				if g.ArchiveRole == 0 {
					err = bot.GenericError(c.FnName, "getting archive role", "`archive_role` not set, use `archive role [role id]`")
				}
				return g, "ChannelCommand: check archive permission"
			})

			if err != nil {
				return err
			}

			channel, err := bot.Client.Channel(c.E.ChannelID)
			if err != nil {
				return err
			}

			overwrites := make([]discord.Overwrite, 0)
			var data api.ModifyChannelData

			bot.GuildContext(c.E.GuildID, func(g *bot.GuildConfig) (*bot.GuildConfig, string) {
				// Copy everything except the archive and @everyone roles to overwrites
				for _, overwrite := range channel.Overwrites {
					id := int64(overwrite.ID)
					if id != int64(c.E.GuildID) && id != g.ArchiveRole {
						overwrites = append(overwrites, overwrite)
						break
					}
				}

				overwrites = append(
					overwrites,
					discord.Overwrite{
						ID:   discord.Snowflake(c.E.GuildID),
						Type: discord.OverwriteRole,
						Deny: discord.PermissionViewChannel,
					},
					discord.Overwrite{
						ID:    discord.Snowflake(g.ArchiveRole),
						Type:  discord.OverwriteRole,
						Allow: discord.PermissionViewChannel,
					},
				)
				data = api.ModifyChannelData{Overwrites: &overwrites, CategoryID: discord.ChannelID(g.ArchiveCategory)}

				return g, "ChannelCommand: create overwrites data"
			})

			err = bot.Client.ModifyChannel(c.E.ChannelID, data)
			if err != nil {
				return err
			} else {
				_, err = cmd.SendEmbed(c.E, "Channel Archive", "Successfully archived channel", bot.SuccessColor)
				return err
			}
		default:
			return defaultResponse()
		}
	case "slow":
		if err := cmd.HasPermission(c, cmd.PermChannels); err != nil {
			return err
		}

		seconds, _ := cmd.ParseInt64Arg(c.Args, 2)
		channelID := c.E.ChannelID

		if channel, err := cmd.ParseChannelArg(c.Args, 2); err == nil {
			channelID = discord.ChannelID(channel)
			seconds, _ = cmd.ParseInt64Arg(c.Args, 3)
		}

		if seconds < 0 { // normalize to 0-21600
			seconds = 0
		} else if seconds > 21600 {
			seconds = 21600
		}

		data := api.ModifyChannelData{UserRateLimit: option.NewNullableUint(uint(seconds))}
		if err := bot.Client.ModifyChannel(channelID, data); err != nil {
			return err
		} else {
			message := fmt.Sprintf("Set slowmode to %v!", util.FormattedTime(seconds))
			if seconds == 0 {
				message = "Cleared slowmode!"
			}
			if channelID != c.E.ChannelID {
				message = fmt.Sprintf("Set slowmode in <#%v> to %v!", channelID, util.FormattedTime(seconds))
				if seconds == 0 {
					message = fmt.Sprintf("Cleared slowmode in <#%v>!", channelID)
				}
			}
			_, err = cmd.SendEmbed(c.E, "Channel Slow", message, bot.SuccessColor)
			return err
		}
	default:
		return defaultResponse()
	}
}

func PermissionCommand(c bot.Command) error {
	arg1, _ := cmd.ParseStringArg(c.Args, 1, true)

	switch arg1 {
	case "give":
		if err := cmd.HasPermission(c, cmd.PermPermissions); err != nil {
			return err
		}

		permission, argErr := cmd.ParseStringArg(c.Args, 2, true)
		if argErr != nil {
			return argErr
		}
		id, argErr := cmd.ParseUserArg(c.Args, 3)
		if argErr != nil {
			return argErr
		}

		if err := cmd.GivePermission(c, permission, id); err != nil {
			return err
		} else {
			_, err = cmd.SendEmbed(c.E,
				"Permissions",
				"Successfully gave "+util.GetUserMention(id)+" permission to use \""+permission+"\"",
				bot.SuccessColor)
			return err
		}
	case "op":
		if err := cmd.HasPermission(c, cmd.PermPermissions); err != nil {
			return err
		}

		color := bot.SuccessColor
		errs := 0
		responses := make([]string, 0)

		for _, permission := range cmd.Permissions {
			if err := cmd.GivePermission(c, permission.String(), int64(c.E.Author.ID)); err != nil {
				responses = append(responses, fmt.Sprintf("⛔ Failed to give \"%s\" permission:%s\n", permission, err.Error()))
				errs += 1
			} else {
				responses = append(responses, fmt.Sprintf("✅ Granted \"%s\" permission\n", permission))
			}
		}

		if errs == len(cmd.Permissions) {
			color = bot.ErrorColor
		} else if errs > 0 {
			color = bot.WarnColor
		}

		_, err := cmd.SendEmbed(c.E,
			"Permissions",
			strings.Join(responses, "\n"),
			color)

		return err
	default:
		_, err := cmd.SendEmbed(c.E,
			"Permissions",
			"Available arguments are:\n- `give` <permission> <user>\n- `op`",
			bot.DefaultColor)
		return err
	}
}

func ProfilePicCommand(c bot.Command) error {
	self := false
	id, argErr := cmd.ParseInt64Arg(c.Args, 1)
	if argErr != nil {
		id, argErr = cmd.ParseUserArg(c.Args, 1)
		if argErr != nil {
			self = true
			id = int64(c.E.Author.ID)
		}
	}

	url := ""
	name := c.E.Author.Username

	// if command is not being run inside a DM
	if c.E.Member != nil {
		name = c.E.Member.Nick
	}

	if self {
		url = c.E.Author.AvatarURLWithType(discord.AutoImage)
	} else {
		user, err := bot.Client.User(discord.UserID(id))
		if err != nil {
			return err
		}
		url = user.AvatarURLWithType(discord.AutoImage)
		name = user.Username
	}

	url += "?size=2048"

	e := discord.Embed{
		Title: name,
		URL:   url,
		Image: &discord.EmbedImage{URL: url},
		Color: bot.WhiteColor,
	}
	_, err := cmd.SendCustomEmbed(c.E.ChannelID, e)
	return err
}

func SudoCommand(c bot.Command) error {
	if err := cmd.HasPermission(c, cmd.PermOperator); err != nil {
		return err
	}

	arg, _ := cmd.ParseStringArg(c.Args, 1, true)

	// Look for a command alias
	var alias []string
	bot.C.Run(func(cf *bot.Config) {
		if a, ok := cf.OperatorAliases[arg]; ok {
			alias = a
		}
	})

	// TODO: Test privileges, alias, args.
	// Found an alias, execute it as a command and return. This does not allow privilege escalation because c.E is still passed.
	if len(alias) > 0 {
		args, _ := cmd.ParseStringSliceArg(c.Args, 2, -1)
		if len(args) > 0 { // Allow adding additional args.
			alias = append(alias, args...)
		}

		cmd.CommandHandlerWithCommand(c.E, alias[0], alias[1:])
		return nil
	}

	// We didn't find an alias, so instead check the regular command args.
	switch arg {
	case "alias":
		aliasName, argErr := cmd.ParseStringArg(c.Args, 2, true)
		if argErr != nil {
			return argErr
		}

		args, _ := cmd.ParseStringSliceArg(c.Args, 3, -1)
		var err error

		bot.C.Run(func(cf *bot.Config) {
			if cf.OperatorAliases == nil {
				cf.OperatorAliases = make(map[string][]string, 0)
			}

			getAliases := func(arg string) (*discord.Message, error) {
				var err error
				var msg *discord.Message
				if len(cf.OperatorAliases) == 0 {
					msg, err = cmd.SendEmbed(c.E, c.Name+" `alias "+arg+"`", fmt.Sprintf("No aliases are currently set! Use the `%s alias [alias]` command to set an alias.", c.Name), bot.ErrorColor)
				} else {
					aliases := make([]string, 0)
					for name, _ := range cf.OperatorAliases {
						aliases = append(aliases, fmt.Sprintf("- `%s`", name))
					}
					util.SliceSortAlphanumeric(aliases)
					msg, err = cmd.SendEmbed(c.E, c.Name+" `alias "+arg+"`", fmt.Sprintf("The following aliases are currently set:\n%s\n", strings.Join(aliases, "\n")), bot.DefaultColor)
				}

				return msg, err
			}

			switch aliasName {
			case "-l":
				_, err = getAliases("-l")
			case "-r":
				if len(args) > 0 {
					if alias, ok := cf.OperatorAliases[args[0]]; !ok {
						_, err = cmd.SendEmbed(c.E, c.Name+" `alias -r`", fmt.Sprintf("Could not find any alias with the name `%s`!", args[0]), bot.ErrorColor)
					} else {
						_, err = cmd.SendEmbed(c.E, c.Name+" `alias -r`", fmt.Sprintf("```\nalias %s %s\n```", args[0], strings.Join(alias, " ")), bot.ErrorColor)
						delete(cf.OperatorAliases, args[0])
					}
				} else {
					_, err = cmd.SendEmbed(c.E, c.Name+" `alias -r`", "You need to specify which alias to remove!", bot.ErrorColor)
				}
			case "--export":
				if c.E.GuildID.IsValid() {
					_, err = cmd.SendEmbed(c.E, c.Name+" `alias --export`", "Cannot import aliases while in guilds! (You could potentially leak private information).", bot.ErrorColor)
				} else {
					if j, err1 := json.Marshal(cf.OperatorAliases); err1 != nil {
						err = err1
					} else {
						b64 := base64.StdEncoding.EncodeToString(j)
						if len(b64) > 4088 {
							if len(cf.FohToken) == 0 {
								_, err = cmd.SendEmbed(c.E, c.Name+" `alias --export`", "Config is more than 4088 chars but fs-over-http token is not set, cannot upload.", bot.ErrorColor)
							} else {
								msgL, _ := getAliases("--export")
								msgW, _ := cmd.SendEmbed(c.E, c.Name+" `alias --export`", "Config is more than 4088 chars.\nAttempting to upload to fs-over-http", bot.WarnColor)

								cleanupMsg := func(msg *discord.Message, sleep time.Duration) {
									if sleep > 0 {
										time.Sleep(sleep * time.Second)
									}
									_ = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
								}

								go func() {
									cleanupMsg(msgW, 5) // Delete warning after 5 seconds
								}()

								// Create body and writer
								body := &bytes.Buffer{}
								writer := multipart.NewWriter(body)

								// Create form file from b64
								cfgName := fmt.Sprintf("alias-config-%s%v.txt", bot.User.ID, time.Now().UnixMilli())
								part, _ := writer.CreateFormFile("file", cfgName)
								encoder := base64.NewEncoder(base64.StdEncoding, part)

								if _, err1 := encoder.Write(j); err1 != nil {
									_ = writer.Close()
									go cleanupMsg(msgL, 1)
									go cleanupMsg(msgW, 1)
									err = err1
								} else {
									// We HAVE to close the writer on our own before making a request otherwise we will be led on a wild goose chase into the http lib.
									// Please do not try to debug why this fails to close on its own and why a `defer writer.Close()` isn't good enough.
									// For some reason this has to be closed before the form is parsed, I presume because there's a stack that needs to be pushed by it.
									// TIME WASTED HERE: 4 hours on the dot.
									_ = writer.Close()

									// Upload file
									r, _ := http.NewRequest("POST", fmt.Sprintf("%s%s%s", cf.FohPrivateUrl, cf.FohPrivateDir, cfgName), body)
									r.Header.Add("Content-Type", writer.FormDataContentType())
									r.Header.Set("Auth", cf.FohToken)

									if err1 := r.ParseForm(); err1 != nil {
										go cleanupMsg(msgL, 1)
										go cleanupMsg(msgW, 1)
										err = err1
									} else {
										if content, res, err1 := util.RequestUrlReq(r); err1 != nil {
											go cleanupMsg(msgL, 1)
											go cleanupMsg(msgW, 1)
											err = err1
										} else if res != nil && res.StatusCode != 200 {
											go cleanupMsg(msgL, 1)
											go cleanupMsg(msgW, 1)
											_, err = cmd.SendEmbed(c.E, c.Name+" `alias --export`", fmt.Sprintf("Config is more than 4088 chars.\nFailed to upload with the following status:\n```\n%v: %s\n```", res.StatusCode, content), bot.ErrorColor)
										} else {
											_, err = cmd.SendEmbed(c.E, c.Name+" `alias --export`", fmt.Sprintf("Config is more than 4088 chars.\nUploaded to %s%s%s", cf.FohPublicUrl, cf.FohPublicDir, cfgName), bot.SuccessColor)
										}
									}
								}
							}
						} else {
							_, _ = getAliases("--export")
							_, err = cmd.SendEmbed(c.E, c.Name+" `alias --export`", fmt.Sprintf("```\n%s\n```", b64), bot.SuccessColor)
						}
					}
				}
			case "--import":
				if c.E.GuildID.IsValid() {
					var err1 error
					if len(args) > 0 {
						err1 = bot.Client.DeleteMessage(c.E.ChannelID, c.E.ID, "Removing potentially sensitive information (`# alias --import`)")
					}

					embed := cmd.MakeEmbed(c.Name+" `alias --import`", "Cannot import aliases while in guilds! (You could potentially leak private information).", bot.ErrorColor)
					if err1 != nil {
						embed.Description += " \nFailed to delete original message: " + err1.Error()
					}

					_, err = cmd.SendCustomEmbed(c.E.ChannelID, embed)
				} else {
					if len(args) == 0 {
						_, err = cmd.SendEmbed(c.E, c.Name+" `alias --import`", "You need to specify a `base64` alias config to import!", bot.ErrorColor)
					} else {
						var b64 []byte

						// Get a config from a URL
						urlMatch := cmd.UrlRegex.FindStringSubmatch(args[0])
						log.Printf("urlMatch: %s\n", urlMatch)
						if len(urlMatch) != -1 {
							go func() {
								msg, _ := cmd.SendEmbed(c.E, c.Name+" `alias --import`", "Found URL as parameter, attempting to load from URL", bot.WarnColor)
								time.Sleep(5 * time.Second)
								_ = bot.Client.DeleteMessage(msg.ChannelID, msg.ID, "cleaning up log msg")
							}()

							// If all values are set, update request URL
							if strings.HasPrefix(urlMatch[0], cf.FohPublicUrl+cf.FohPublicDir) &&
								util.SlicesCondition([]string{cf.FohToken, cf.FohPublicUrl, cf.FohPublicDir, cf.FohPrivateUrl, cf.FohPrivateDir},
									// Ensure that each variable is not empty
									func(c string) bool {
										return len(c) > 0
									},
								) {
								// Replace beginning of public URL with private when requesting, if cf.FohToken and all other variables are set
								urlMatch[0] = cf.FohPrivateUrl + cf.FohPrivateDir + strings.TrimPrefix(urlMatch[0], cf.FohPublicUrl+cf.FohPublicDir)
							}

							log.Printf("urlMatch: %s\n", urlMatch)

							// Request b64 content from URL
							if content, _, err1 := util.RequestUrlFn(urlMatch[0], http.MethodGet, func(req *http.Request) {
								if len(cf.FohToken) > 0 {
									req.Header.Add("Auth", cf.FohToken)
								}
							}); err1 == nil {
								b64 = content
							} else {
								err = err1
							}
						} else { // Default to reading base64 from the message
							if content, err1 := base64.StdEncoding.DecodeString(strings.Join(args, "")); err1 == nil {
								b64 = content
							} else {
								err = err1
							}
						}

						// Parse config, either from a message or a URL, and import it
						if err == nil {
							j := make([]byte, base64.StdEncoding.DecodedLen(len(b64)))
							if _, err1 := base64.StdEncoding.Decode(j, b64); err1 == nil {
								var aliases map[string][]string
								if err1 := json.Unmarshal(j, &aliases); err1 != nil {
									err = err1
								} else {
									cf.OperatorAliases = aliases
									if _, err1 = cmd.SendEmbed(c.E, c.Name+" `alias --import`", "Imported aliases!", bot.SuccessColor); err1 != nil {
										err = err1
									} else {
										_, err = getAliases("--import")
									}
								}
							} else {
								err = err1
							}
						}
					}
				}
			default:
				if len(args) == 0 {
					if alias, ok := cf.OperatorAliases[aliasName]; !ok {
						_, err = cmd.SendEmbed(c.E, c.Name+" `alias`", fmt.Sprintf("Could not find any alias with the name `%s`!", aliasName), bot.ErrorColor)
					} else {
						_, err = cmd.SendEmbed(c.E, c.Name+" `alias`", fmt.Sprintf("```\nalias %s %s\n```", aliasName, strings.Join(alias, " ")), bot.DefaultColor)
					}
				} else {
					cf.OperatorAliases[aliasName] = args
					_, err = cmd.SendEmbed(c.E, c.Name+" `alias`", fmt.Sprintf("```\nalias %s %s\n```", aliasName, strings.Join(args, " ")), bot.SuccessColor)
				}
			}
		})
		return err
	case "-h":
		_, err := cmd.SendEmbed(c.E,
			c.Name,
			"Available arguments are:\n- `alias <name> [command]`\n- `alias -r <name>`\n- `alias -l|-h|--export|--import`",
			bot.DefaultColor)
		return err
	default: // Default to running a bash shell
		if args, err := cmd.ParseStringSliceArg(c.Args, 1, -1); err != nil {
			return err
		} else {
			if res, err := httpBashRequests.Run(strings.Join(args, " ") + " 2>&1"); err != nil {
				return err
			} else {
				_, err := cmd.SendEmbed(c.E, "", fmt.Sprintf("```\n%s\n```", util.TailLinesLimit(string(res), 2040)), bot.DefaultColor)
				return err
			}
		}
	}
}
