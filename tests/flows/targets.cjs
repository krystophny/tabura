const targetDefinitions = {
  sloppad_circle_dot: {
    category: 'circle',
    platforms: {
      web: '#sloppad-circle-dot',
      ios: 'sloppad_circle_dot',
      android: 'sloppad_circle_dot',
    },
  },
  sloppad_circle_segment_pointer: {
    category: 'circle',
    platforms: {
      web: '[data-segment="pointer"]',
      ios: 'sloppad_circle_pointer',
      android: 'sloppad_circle_pointer',
    },
  },
  sloppad_circle_segment_highlight: {
    category: 'circle',
    platforms: {
      web: '[data-segment="highlight"]',
      ios: 'sloppad_circle_highlight',
      android: 'sloppad_circle_highlight',
    },
  },
  sloppad_circle_segment_ink: {
    category: 'circle',
    platforms: {
      web: '[data-segment="ink"]',
      ios: 'sloppad_circle_ink',
      android: 'sloppad_circle_ink',
    },
  },
  sloppad_circle_segment_text_note: {
    category: 'circle',
    platforms: {
      web: '[data-segment="text_note"]',
      ios: 'sloppad_circle_text_note',
      android: 'sloppad_circle_text_note',
    },
  },
  sloppad_circle_segment_prompt: {
    category: 'circle',
    platforms: {
      web: '[data-segment="prompt"]',
      ios: 'sloppad_circle_prompt',
      android: 'sloppad_circle_prompt',
    },
  },
  sloppad_circle_segment_dialogue: {
    category: 'circle',
    platforms: {
      web: '[data-segment="dialogue"]',
      ios: 'sloppad_circle_dialogue',
      android: 'sloppad_circle_dialogue',
    },
  },
  sloppad_circle_segment_meeting: {
    category: 'circle',
    platforms: {
      web: '[data-segment="meeting"]',
      ios: 'sloppad_circle_meeting',
      android: 'sloppad_circle_meeting',
    },
  },
  sloppad_circle_segment_silent: {
    category: 'circle',
    platforms: {
      web: '[data-segment="silent"]',
      ios: 'sloppad_circle_silent',
      android: 'sloppad_circle_silent',
    },
  },
  canvas_viewport: {
    category: 'circle',
    platforms: {
      web: '#canvas-viewport',
      ios: 'canvas_viewport',
      android: 'canvas_viewport',
    },
  },
  indicator_border: {
    category: 'indicator',
    platforms: {
      web: '#indicator-border',
      ios: 'indicator_border',
      android: 'indicator_border',
    },
  },
  indicator_simulate_recording: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-simulate-recording',
    },
  },
  indicator_simulate_working: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-simulate-working',
    },
  },
  indicator_override_clear: {
    category: 'indicator',
    kind: 'test_hook',
    platforms: {
      web: '#indicator-override-clear',
    },
  },
};

const requiredCoverageTargets = [
  'sloppad_circle_dot',
  'sloppad_circle_segment_pointer',
  'sloppad_circle_segment_highlight',
  'sloppad_circle_segment_ink',
  'sloppad_circle_segment_text_note',
  'sloppad_circle_segment_prompt',
  'sloppad_circle_segment_dialogue',
  'sloppad_circle_segment_meeting',
  'sloppad_circle_segment_silent',
  'canvas_viewport',
  'indicator_border',
];

const requiredIndicatorStates = ['idle', 'listening', 'paused', 'recording', 'working'];

function getTargetDefinition(target) {
  return targetDefinitions[target] || null;
}

function getTargetPlatforms(target) {
  const definition = getTargetDefinition(target);
  return definition ? Object.keys(definition.platforms) : [];
}

function getWebSelector(target) {
  const definition = getTargetDefinition(target);
  return definition && definition.platforms.web ? definition.platforms.web : null;
}

module.exports = {
  getTargetDefinition,
  getTargetPlatforms,
  getWebSelector,
  requiredCoverageTargets,
  requiredIndicatorStates,
  targetDefinitions,
};
