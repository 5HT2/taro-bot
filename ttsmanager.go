package main

import (
	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"context"
	"fmt"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/voice"
	texttospeechpb "google.golang.org/genproto/googleapis/cloud/texttospeech/v1"
	"io"
)

func RunTTS(c Command) error {
	voice.AddIntents(discordClient.Gateway)

	s, err := state.New("Bot " + config.BotToken)
	if err != nil {
		return err
	}
	voice.AddIntents(s.Gateway)

	if err := s.Open(context.Background()); err != nil {
		return err
	}
	defer s.Close()

	v, err := voice.NewSession(s)
	if err != nil {
		return err
	}

	if err := v.JoinChannel(c.e.GuildID, c.e.ChannelID, false, true); err != nil {
		return err
	}
	defer v.Leave()

	return nil
}

// ListVoices lists the available text to speech voices.
func ListVoices(w io.Writer) error {
	ctx := context.Background()

	client, err := texttospeech.NewClient(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	// Performs the list voices request.
	resp, err := client.ListVoices(ctx, &texttospeechpb.ListVoicesRequest{})
	if err != nil {
		return err
	}

	for _, voice := range resp.Voices {
		// Display the voice's name. Example: tpc-vocoded
		fmt.Fprintf(w, "Name: %v\n", voice.Name)

		// Display the supported language codes for this voice. Example: "en-US"
		for _, languageCode := range voice.LanguageCodes {
			fmt.Fprintf(w, "  Supported language: %v\n", languageCode)
		}

		// Display the SSML Voice Gender.
		fmt.Fprintf(w, "  SSML Voice Gender: %v\n", voice.SsmlGender.String())

		// Display the natural sample rate hertz for this voice. Example: 24000
		fmt.Fprintf(w, "  Natural Sample Rate Hertz: %v\n",
			voice.NaturalSampleRateHertz)
	}

	return nil
}
